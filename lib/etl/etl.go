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

package etl

import (
	"container/list"
	"github.com/fatima-go/fatima-core/lib"
	"github.com/fatima-go/fatima-log"
	"math"
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

type SimpleETL interface {
	Process()
}

type SimpleETLBuilder interface {
	SetExtractor(func(ExtractionDeliver) error) SimpleETLBuilder
	SetLogger(Logger) SimpleETLBuilder
	SetTransformQueueSize(int) SimpleETLBuilder
	SetTransformer(int, func(interface{}, Loader)) SimpleETLBuilder
	SetLoader(func(interface{})) SimpleETLBuilder
	Build() (SimpleETL, error)
}

type ExtractionDeliver interface {
	Deliver(interface{})
}

type Loader interface {
	Load(interface{})
}

type Logger interface {
	Printf(string, ...interface{})
}

func NewSimpleETLBuilder() SimpleETLBuilder {
	builder := new(dataFetchBuilder)
	builder.transformQueueSize = math.MaxInt16
	builder.transformWorkerSize = runtime.NumCPU()
	return builder
}

type dataFetchBuilder struct {
	extractor           func(ExtractionDeliver) error
	logger              Logger
	transformFunc       func(interface{}, Loader)
	loader              func(interface{})
	transformWorkerSize int
	transformQueueSize  int
}

func (f *dataFetchBuilder) SetExtractor(extractor func(ExtractionDeliver) error) SimpleETLBuilder {
	f.extractor = extractor
	return f
}

func (f *dataFetchBuilder) SetLogger(logger Logger) SimpleETLBuilder {
	f.logger = logger
	return f
}

func (f *dataFetchBuilder) SetTransformer(workerSize int, transformFunc func(interface{}, Loader)) SimpleETLBuilder {
	if workerSize > 0 {
		f.transformWorkerSize = workerSize
	}
	f.transformFunc = transformFunc
	return f
}

func (f *dataFetchBuilder) SetTransformQueueSize(size int) SimpleETLBuilder {
	if size > 0 {
		f.transformQueueSize = size
	}
	return f
}

func (f *dataFetchBuilder) SetLoader(loader func(interface{})) SimpleETLBuilder {
	f.loader = loader
	return f
}

func (f *dataFetchBuilder) Build() (SimpleETL, error) {
	etl := new(simpleETL)
	etl.extractFunc = f.extractor
	etl.logger = f.logger
	etl.loaderFunc = f.loader
	etl.transformFunc = f.transformFunc
	etl.ingestList = list.New()
	etl.ingestFinish = false
	etl.dataChan = make(chan interface{}, math.MaxInt16)
	etl.executor = lib.NewExecutorBuilder(etl.transform).
		SetQueueSize(f.transformQueueSize).
		SetWorkerSize(f.transformWorkerSize).
		Build()

	return etl, nil
}

type simpleETL struct {
	executor      lib.Executor
	extractFunc   func(ExtractionDeliver) error
	transformFunc func(interface{}, Loader)
	dataChan      chan interface{}
	loaderFunc    func(interface{})
	logger        Logger
	loadWg        sync.WaitGroup
	deliverWg     sync.WaitGroup
	gofuncWg      sync.WaitGroup
	loadCount     uint32
	ingestList    *list.List
	ingestFinish  bool
}

func (d *simpleETL) transform(val interface{}) {
	d.transformFunc(val, d)
}

func (d *simpleETL) Process() {
	defer func() {
		d.ingestList = nil
		d.executor.Close()
		close(d.dataChan)

		if r := recover(); r != nil {
			log.Error("process panic : %v", r)
			if d.logger != nil {
				d.logger.Printf("process panic : %v\n", r)
			}
			log.Error("%s", string(debug.Stack()))
			return
		}
	}()

	d.gofuncWg.Add(2)
	go d.startLoading()
	go d.startDeliverToTransform()

	// wait for a second (go func started...)
	d.gofuncWg.Wait()

	startMillis := lib.CurrentTimeMillis()
	log.Info("start loading....")
	if d.logger != nil {
		d.logger.Printf("%s", "start loading....\n")
	}

	d.startExtracting()

	d.loadWg.Wait()

	log.Info("ETL finish. total %d/%d done : %s", d.executor.Count(), d.loadCount, lib.ExpressDuration(startMillis))
	if d.logger != nil {
		d.logger.Printf("ETL finish. total %d/%d done : %s\n", d.executor.Count(), d.loadCount, lib.ExpressDuration(startMillis))
	}
}

func (d *simpleETL) startExtracting() {
	log.Info("start extracting with concurrent extractor")
	if d.logger != nil {
		d.logger.Printf("%s", "start extracting with concurrent extractor\n")
	}

	startMillis := lib.CurrentTimeMillis()
	err := d.extractFunc(d)
	if err != nil {
		log.Warn("fail to extract : %s", err)
		if d.logger != nil {
			d.logger.Printf("fail to extract : %s\n", err)
		}
		return
	}

	d.ingestFinish = true
	d.deliverWg.Wait()
	log.Warn("waiting extract done. %s", lib.ExpressDuration(startMillis))
	if d.logger != nil {
		d.logger.Printf("waiting extract done. %s", lib.ExpressDuration(startMillis))
	}

	d.executor.Wait()

	elapsed := lib.ExpressDuration(startMillis)
	log.Warn("extract and transform finished. %d record. %s", d.executor.Count(), elapsed)
	if d.logger != nil {
		d.logger.Printf("extract and transform finished. %d record. %s\n", d.executor.Count(), elapsed)
	}
}

func (d *simpleETL) Deliver(v interface{}) {
	d.ingestList.PushBack(v)
}

func (d *simpleETL) startLoading() {
	d.gofuncWg.Done()
	for elem := range d.dataChan {
		d.loaderFunc(elem)
		d.loadWg.Done()
	}
}

func (d *simpleETL) startDeliverToTransform() {
	log.Info("startDeliverToTransform...")
	d.deliverWg.Add(1)
	defer func() {
		d.deliverWg.Done()
	}()

	d.gofuncWg.Done()
	for true {
		if d.ingestList.Len() == 0 && d.ingestFinish {
			return
		}

		elem := d.ingestList.Front()
		if elem == nil {
			time.Sleep(time.Millisecond * 100)
			continue
		}

		d.executor.Insert(elem.Value)
		d.ingestList.Remove(elem)
	}
}

func (d *simpleETL) Load(v interface{}) {
	d.loadWg.Add(1)
	atomic.AddUint32(&d.loadCount, 1)
	d.dataChan <- v
}
