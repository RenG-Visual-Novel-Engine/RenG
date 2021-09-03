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
a = 1 + 1
b = 2 - 2
c = add(a, b)
```

```
ch_example = Character("ex", color="#c8ffc8")

label start {
    ch_example "Hello, World!"
}
```

<br>

# 개발 참가 전 알아야 할 사항들

- Ren'G 엔진은 cgo를 이용하므로 gcc 혹은 clang 등의 C언어 컴파일 환경이 필요합니다. 윈도우 유저시라면 mingw-w64를 사용할 것을 권장합니다.
- 각각 파일에 존재하는 main.go 파일들은 테스트를 위해 만들어 둔 파일입니다. 지우시고 개발하셔도 됩니다.
- 1회 이상 엔진 개발에 기여할시 RenG-Visual-Novel-Engine 팀에 초대됩니다.

<br><br>

- ## 엔진의 전체적인 구조입니다.

<br><br>

<img src="https://user-images.githubusercontent.com/77112874/131224110-d66b9175-ca1d-406f-b331-2da5f58d605a.jpg"></img>

<br>

# 시작

```
git clone "https://github.com/alvin1007/RenG"
```

- 파일 위치는 "Go/src/github.com/alvin1007/RenG"에 해두는 것이 가장 좋습니다.

# 커밋 규칙

```
git commit -m "<본인의 계정 이름> : <수정 내용>"
```

## 디스코드
<a href="https://discord.gg/JkkvP6U9qk" terget="_blank">
<img src="https://img.shields.io/badge/-Discord-5865F2?logo=Discord&logoColor=white&style=flat"/>
</a>