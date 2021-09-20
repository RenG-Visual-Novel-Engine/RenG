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

# 문법

- ## Example

```
def add(a, b) {
    return a + b
}

define ch_example1 = Character("ex1", color="#c8ffc8")
define ch_example2 = Character("ex2", color="#ffffff")

label start {
    ch_example1 "안녕? 혹시 1 + 1이 뭐야?"

    result = add(1, 1)

    ch_example2 "답은 [result]야!"
}
```

## 자료형
---
|자료형|실제 처리되는 타입|
|----|--------------------|
|null|null|
|int|int64|
|float|float64|
|string|string|
|bool|bool|
|list|다중 타입 지원|

<br>

## 연산자
---
|연산자|처리되는 연산|지원되는 자료형|
|:-:|:-:|:-|
|+|덧셈 (문자열은 서로 연결해줌)|int, float, string|
|-|뺄셈|int, float|
|*|곱셈|int, float|
|/|나누고 몫을 반환 (정수일 경우 소수점은 무시됨)|int, float|
|%|나누고 나머지를 반환|int|

<br>

|비트 연산자|처리되는 연산|지원되는 자료형|
|:-:|:-:|:-|
|&|AND 비트 연산|int|
|&#124;|OR 비트 연산|int|
|^|XOR 비트 연산|int|

<br>

|논리 연산자|처리되는 연산|지원되는 자료형|
|:-:|:-:|:-|
|==|오른쪽 값과 왼쪽 값이 같은가?|int, float(정확도는 떨어질 수 있습니다.), string, bool|
|!=|오른쪽 값과 왼쪽 값이 다른가?|int, float, string, bool|
|>|오른쪽 값이 더 큰가?|int, float|
|<|왼쪽 값이 더 큰가?|int, float|
|>=|오른쪽 값이 더 크거나 같은가?|int, float|
|<=|왼쪽 값이 더 크거나 같은가?|int, float|
|!|NOT 연산|bool|

<br>

|대입 연산자|처리되는 연산|지원되는 자료형|
|:-:|:-:|:-|
|=|오른쪽 값을 왼쪽 변수에 대입|int, float, string, bool, list|
|+=|왼쪽 값과 오른쪽 값을 더한 뒤, 왼쪽 변수에 대입|int, float, (string 곧 추가)|
|-=|왼쪽 값과 오른쪽 값을 뺸 뒤, 왼쪽 변수에 대입|int, float|
|*=|왼쪽 값과 오른쪽 값을 곱한 뒤, 왼쪽 변수에 대입|int, float|
|/=|왼쪽 값과 오른쪽 값을 나눈 뒤, 그 몫을 왼쪽 변수에 대입|int, float|
|%=|왼쪽 값과 오른쪽 값을 나눈 뒤, 그 나머지를 왼쪽 변수에 대입|int|

<br>

|전위 연산자|처리되는 연산|지원되는 자료형|
|:-:|:-:|:-|
|++|오른쪽 변수를 1 증가시킴|int|
|--|오른쪽 변수를 1 감소시킴|int|
|-|오른쪽 양수를 음수로 전환|int, float|

<br>

## 지원되는 구문
---
> if 문
```
if (<Condition>) {
    ...
} elif (<Condition>) {
    ...
} else {
    ...
}
```
> for 문
```
for (<Expression>; <Condition>; <Expression>) {
    ...
}
```
> while 문
```
while (<Condition>) {
    ...
}
```
> function, return 문
```
def <Idenifier>(<Arg>, <Arg>, <Arg>, ... ) {
    ...
    return <Value>
}
```
> label 문
```
label <Identifier> {
    ...
}
```

<br>

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

# 라이선스

- 본 저작물은 MIT LICENSE를 따르고 있습니다.
- 본 저작물에서 사용한 제주고딕체는 공공누리 제1유형에 의거하여 [해당](https://www.jeju.go.kr/jeju/symbol/font/gothic.htm) 부분을 클릭하여 다운로드 받을 수 있음을 알려드립니다.