# fatima-core #
fatima-core 프로젝트는 [golang 언어를 사용한 손쉬운 프로그램 개발](https://github.com/fatima-go/.github/blob/main/profile/development.md)을 도와주는 프레임워크를 제공한다.

## 개발과 테스트 ##
fatima-core 의 개발과 테스트는 내가 변경한 fatima-core 코드에 대해 pseudo version을 생성하여 tagging한다. 이후에 해당 tag 를 이용하여 별도의 프로그램에서 테스트를 진행하는 방식을 사용한다.
참고로 별도 개발 브랜치에서 작업 후 해당 브랜치를 테스트하기 위해서는 golang module version numbering 정책을 따라간다.
- [Go Big With Pseudo-Versions and GoCenter](https://jfrog.com/blog/go-big-with-pseudo-versions-and-gocenter/)
- [Module version numbering](https://go.dev/doc/modules/version-numbers)

먼저 (현재 개발중인) 소스의 태깅을 위해 현재 브랜치의 가장 최근 커밋 정보를 적절한 포맷으로 출력할 수 있도록 변수를 정의한다
```shell
TZ=UTC git --no-pager show \
   --quiet \
   --abbrev=12 \
   --date='format-local:%Y%m%d%H%M%S' \
   --format="%cd-%h"
```
다음으로 모듈의 pseudo 버저닝을 위해 적절한 버전과 커밋 정보를 조합하여 태깅하고 push를 한다.
```shell
% TZ=UTC git --no-pager show \
   --quiet \
   --abbrev=12 \
   --date='format-local:%Y%m%d%H%M%S' \
   --format="%cd-%h"
20231004081555-6f0f6cc31723
% git tag v0.0.1-20231004081555-6f0f6cc31723
% git push origin v0.0.1-20231004081555-6f0f6cc31723
Total 0 (delta 0), reused 0 (delta 0), pack-reused 0
To https://github.com/fatima-go/fatima-core.git
 * [new tag]         v0.0.1-20231004081555-6f0f6cc31723 -> v0.0.1-20231004081555-6f0f6cc31723
```

# release #
- [release history](./RELEASE.md)

