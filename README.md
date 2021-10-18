<div align='center'>
<h1>Ren'G</h1>
<h3>Visual Novel Engine</h3>
<h3>In Go</h3>
</div>
<br><br><br><br>

# 추구하는 것
- 빠른 속도
- 간단한 문법
- 훌륭한 소리
- 아름답고 간편한 디자인

<br><br>

# 라이선스

## RenG
MIT License

Copyright (c) 2021 RenG-Visual-Novel-Engine

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

<br>

## Libraries

|Name|역할|라이선스|
|----|----|----|
|SDL2|메인 그래픽 출력|zlib License,|
|FFmpeg|동영상 인코딩, 디코딩|GNU Lesser General Public License|

## Font
- 본 저작물에서 사용한 제주고딕체는 공공누리 제1유형에 의거하여 [해당](https://www.jeju.go.kr/jeju/symbol/font/gothic.htm) 부분을 클릭하여 다운로드 받을 수 있음을 알려드립니다.

<br><br>

# 디스코드
<a href="https://discord.gg/JkkvP6U9qk" terget="_blank">
<img src="https://img.shields.io/badge/-Discord-5865F2?logo=Discord&logoColor=white&style=flat"/>
</a>

<br><br>

# 개발 참가 전 알아야 할 사항들

- Ren'G 엔진은 cgo를 이용하므로 gcc 혹은 clang 등의 C언어 컴파일 환경이 필요합니다. 윈도우 유저시라면 mingw-w64를 사용할 것을 권장합니다.
- 각각 파일에 존재하는 ~ _test.go 파일들은 테스트를 위해 만들어 둔 파일입니다.
- 1회 이상 엔진 개발에 기여할시 RenG-Visual-Novel-Engine 팀에 초대됩니다.

<br><br>

- ## 엔진의 전체적인 구조입니다.

<br><br>

<img src="https://user-images.githubusercontent.com/77112874/131224110-d66b9175-ca1d-406f-b331-2da5f58d605a.jpg"></img>

<br>

# 시작

```
git clone "https://github.com/RenG-Visual-Novel-Engine/RenG"
```

- 파일 위치는 "{GOPATH}/src/RenG"에 해두는 것이 가장 좋습니다.

# 커밋 규칙

```
git commit -m "<본인의 계정 이름> : <수정 내용>"
```