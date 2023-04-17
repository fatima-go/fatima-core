/*
 * Copyright 2023 github.com/fatima-go
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * @project fatima-core
 * @author dave
 * @date 23. 4. 12. 오후 1:41
 */

package runtime

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/fatima-go/fatima-core"
	"github.com/fatima-go/fatima-log"
	"os"
	"path/filepath"
	"reflect"
	"runtime/debug"
	"strconv"
	"strings"
	"syscall"
	"unicode"
)

const (
	COMMAND_MENU = iota
	COMMAND_CALL
	COMMAND_TEXT
)

const (
	COMMAND_COMMON_QUIT   = "q"
	COMMAND_COMMON_REDO   = "r"
	COMMAND_COMMON_GOBACK = "b"
)

const (
	TYPE_INT = iota
	TYPE_STRING
	TYPE_BOOL
	TYPE_FLOAT
)

type CommandType int
type ParameterType int

var uiSet = &UserInteractionSet{}
var currentStage StageExecutor

type UserInteractionSet struct {
	controller     interface{}
	common         Common
	stages         map[string]Stage
	stageChain     []StageExecutor
	lastExecutions []reflect.Value
}

func (u *UserInteractionSet) start() {
	u.stageChain = make([]StageExecutor, 0)
	u.lastExecutions = make([]reflect.Value, 0)

	currentStage = u.stages["startup"]
	u.stageChain = append(u.stageChain, currentStage)
	for {
		cont := u.getCurrentStage().Execute(u.getCurrentStage().AskInteraction())
		if !cont {
			break
		}
	}

	syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
}

// prepare Method
func (u *UserInteractionSet) prepareMethod(funcName string) error {
	u.lastExecutions = nil
	u.lastExecutions = make([]reflect.Value, 0)

	method := reflect.ValueOf(u.controller).MethodByName(reformToFuncName(funcName))
	if !method.IsValid() {
		return errors.New("not found function")
	}

	u.lastExecutions = append(
		uiSet.lastExecutions,
		method)

	return nil
}

func (u *UserInteractionSet) goBack() {
	l := len(u.stageChain)
	if l < 2 {
		return
	}

	u.stageChain = u.stageChain[:l-1]
}

func (u *UserInteractionSet) goNext(next StageExecutor) {
	u.stageChain = append(u.stageChain, next)
}

func (u *UserInteractionSet) getCurrentStage() StageExecutor {
	l := len(u.stageChain)
	return u.stageChain[l-1]
}

type StageExecutor interface {
	AskInteraction() string
	Execute(userEnter string) bool
}

type Common struct {
	Keywords []Keyword `xml:"keyword"`
}

func (c Common) String() string {
	var buff bytes.Buffer
	for _, v := range c.Keywords {
		buff.WriteString("[")
		buff.WriteString(v.Command)
		buff.WriteString("] ")
		buff.WriteString(v.Text)
		buff.WriteString("\n")
	}

	return buff.String()
}

type Keyword struct {
	Command string `xml:"value,attr"`
	Text    string `xml:",cdata"`
}

type Item struct {
	commandType CommandType `xml:"-"`
	Category    string      `xml:"category,attr"`
	Key         string      `xml:"-"`
	Signature   string      `xml:"sig,attr,omitempty"`
	Text        string      `xml:",cdata"`
}

func (i Item) String() string {
	return fmt.Sprintf("commandType=%s, category=%s, key=%s, sig=%s, text=%s",
		i.commandType, i.Category, i.Key, i.Signature, i.Text)
}

type Parameter struct {
	ptype   ParameterType `xml:"-"`
	Type    string        `xml:"type,attr"`
	Default string        `xml:"default,attr,omitempty"`
	Text    string        `xml:",cdata"`
}

type Stage struct {
	commandType CommandType `xml:"-"`
	Items       []Item      `xml:"item,omitempty"`
	Parameters  []Parameter `xml:"input,omitempty"`
}

func (s Stage) AskInteraction() string {
	if s.commandType == COMMAND_MENU {
		s.printMenu()
	} else {
		s.interactParameters()
	}

	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return scanner.Text()
	}

	return "q"
}

func (s Stage) Execute(userEnter string) bool {
	command := strings.ToLower(userEnter)
	switch command {
	case COMMAND_COMMON_QUIT:
		return false
	case COMMAND_COMMON_REDO:
		execute()
		return true
	case COMMAND_COMMON_GOBACK:
		uiSet.goBack()
	default:
		next := s.findItem(userEnter)
		if next == nil {
			return true
		}
		stage, ok := uiSet.stages[next.Signature]
		if !ok {
			executeBareCommand(next.Signature)
			return true
		}

		if stage.commandType == COMMAND_MENU {
			uiSet.goNext(stage)
		} else if stage.commandType == COMMAND_CALL {
			return executeCommand(next.Signature, stage)
		}
	}
	return true
}

func (s Stage) findItem(value string) *Item {
	for _, v := range s.Items {
		if v.Key == value {
			return &v
		}
	}

	return nil
}

func (s Stage) printMenu() {
	fmt.Printf("\n================\n")
	fmt.Printf("%s\n", s.getGuideText())
	for _, v := range s.Items {
		if v.commandType == COMMAND_TEXT {
			continue
		}
		fmt.Printf("[%s] %s\n", v.Key, v.Text)
	}
	fmt.Printf("\n-------------\n")
	fmt.Printf("%s", uiSet.common)
	fmt.Printf("================\n")
	fmt.Printf("Enter Menu : ")
}

func (s Stage) interactParameters() {

}

func (s Stage) getGuideText() string {
	for _, v := range s.Items {
		if v.commandType == COMMAND_TEXT {
			return v.Text
		}
	}

	return ""
}

func refineStage(stage *Stage) {
	keyIndex := 1
	if len(stage.Items) > 0 {
		stage.commandType = COMMAND_MENU
		for i := 0; i < len(stage.Items); i++ {
			comp := strings.ToLower(stage.Items[i].Category)
			switch comp {
			case "text":
				stage.Items[i].commandType = COMMAND_TEXT
			case "menu":
				stage.Items[i].commandType = COMMAND_MENU
				stage.Items[i].Key = strconv.Itoa(keyIndex)
				keyIndex++
			case "call":
				stage.Items[i].commandType = COMMAND_CALL
				stage.Items[i].Key = strconv.Itoa(keyIndex)
				keyIndex++
			}
		}
	} else {
		stage.commandType = COMMAND_CALL
		for i := 0; i < len(stage.Parameters); i++ {
			comp := strings.ToLower(stage.Parameters[i].Type)
			switch comp {
			case "string":
				stage.Parameters[i].ptype = TYPE_STRING
			case "int":
				stage.Parameters[i].ptype = TYPE_INT
			case "bool":
				stage.Parameters[i].ptype = TYPE_BOOL
			case "float":
				stage.Parameters[i].ptype = TYPE_FLOAT
			default:
				stage.Parameters[i].ptype = TYPE_STRING
			}
		}
	}
}

func executeCommand(funcName string, stage Stage) (ret bool) {
	ret = true
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("**PANIC** while executing...\n", errors.New(fmt.Sprintf("%s", r)))
			fmt.Printf("%s", string(debug.Stack()))
			return
		}
	}()

	answer, ok := askParameters(stage)
	if !ok {
		ret = false
		return
	}

	params, ok := buildParameters(stage, answer)
	if !ok {
		return // finish execution
	}

	err := uiSet.prepareMethod(funcName)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		return
	}
	uiSet.lastExecutions = append(uiSet.lastExecutions, params...)

	execute()

	return
}

func askParameters(stage Stage) ([]string, bool) {
	answer := make([]string, 0)
	if len(stage.Parameters) == 0 {
		return answer, true
	}

	for _, v := range stage.Parameters {
		fmt.Printf("Enter %s (default : %s) = ", v.Text, v.Default)

		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			a := scanner.Text()
			if a == COMMAND_COMMON_QUIT {
				return answer, false
			}
			if len(a) == 0 {
				a = v.Default
			}
			answer = append(answer, a)
		} else {
			fmt.Printf("fail to scan user input...\n")
			return answer, false
		}
	}

	return answer, true
}

func buildParameters(stage Stage, answer []string) ([]reflect.Value, bool) {
	params := make([]reflect.Value, 0)
	for i, v := range stage.Parameters {
		switch v.ptype {
		case TYPE_INT:
			c, err := strconv.Atoi(answer[i])
			if err != nil {
				fmt.Printf("fail to convert %s to int : %s", answer[i], err.Error())
				return params, false
			}
			params = append(params, reflect.ValueOf(c))
		case TYPE_STRING:
			params = append(params, reflect.ValueOf(answer[i]))
		case TYPE_BOOL:
			if strings.ToUpper(answer[i]) == "TRUE" {
				params = append(params, reflect.ValueOf(true))
			} else {
				params = append(params, reflect.ValueOf(false))
			}
		case TYPE_FLOAT:
			c, err := strconv.ParseFloat(answer[i], 64)
			if err != nil {
				fmt.Printf("fail to convert %s to float64 : %s", answer[i], err.Error())
				return params, false
			}
			params = append(params, reflect.ValueOf(c))
		default:
			fmt.Printf("unsupported type : %s", v.Type)
			return params, false
		}
	}

	return params, true
}

func executeBareCommand(funcName string) bool {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("**PANIC** while executing...\n", errors.New(fmt.Sprintf("%s", r)))
			fmt.Printf("%s", string(debug.Stack()))
			return
		}
	}()

	err := uiSet.prepareMethod(funcName)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		return true
	}

	execute()

	return true
}

func execute() {
	if len(uiSet.lastExecutions) == 0 {
		return
	}

	uiSet.lastExecutions[0].Call(uiSet.lastExecutions[1:])
}

func reformToFuncName(funcName string) string {
	if len(funcName) == 0 {
		return funcName
	}

	var buffer bytes.Buffer
	needUpper := true
	for _, c := range funcName {
		if needUpper {
			buffer.WriteRune(unicode.ToUpper(c))
			needUpper = false
			continue
		}

		if c == '_' {
			needUpper = true
			continue
		}

		buffer.WriteRune(c)
	}

	return buffer.String()
}

type UserInteractive struct {
}

func newUserInteractive(controller interface{}) *UserInteractive {
	uiSet.controller = controller
	return &UserInteractive{}
}

func (ui *UserInteractive) Initialize() bool {
	inputFile := filepath.Join(
		process.GetEnv().GetFolderGuide().GetAppFolder(),
		process.GetEnv().GetSystemProc().GetProgramName()+".ui.xml")

	xmlFile, err := os.Open(inputFile)
	if err != nil {
		log.Error("fail to load user interactive xml file : %s", err.Error())
		return false
	}

	defer xmlFile.Close()
	uiSet.stages = make(map[string]Stage)

	decoder := xml.NewDecoder(xmlFile)
	var inElement string
	for {
		t, _ := decoder.Token()
		if t == nil {
			break
		}

		switch se := t.(type) {
		case xml.StartElement:
			inElement = se.Name.Local
			if inElement == "prompt" {
				break
			}

			if inElement == "common" {
				err := decoder.DecodeElement(&uiSet.common, &se)
				if err != nil {
					fmt.Printf("fail to decode xml element : %s", err.Error())
					return false
				}
			} else {
				stage := &Stage{}
				err := decoder.DecodeElement(stage, &se)
				if err != nil {
					fmt.Printf("fail to decode xml element : %s", err.Error())
					return false
				}
				refineStage(stage)
				uiSet.stages[inElement] = *stage
			}
		default:
		}
	}

	uiSet.stageChain = make([]StageExecutor, 0)
	uiSet.lastExecutions = make([]reflect.Value, 0)

	startup, ok := uiSet.stages["startup"]
	if !ok {
		fmt.Printf("not found startup in ui.xml")
		return false
	}
	currentStage = startup
	uiSet.stageChain = append(uiSet.stageChain, currentStage)

	return true
}

func (ui *UserInteractive) Bootup() {
	go func() {
		uiSet.start()
	}()
}

func (ui *UserInteractive) Shutdown() {
}

func (ui *UserInteractive) GetType() fatima.FatimaComponentType {
	return fatima.COMP_GENERAL
}
