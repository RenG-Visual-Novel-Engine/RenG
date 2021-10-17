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

## SDL2
Simple DirectMedia Layer
Copyright (C) 1997-2021 Sam Lantinga <slouken@libsdl.org>

This software is provided 'as-is', without any express or implied
warranty.  In no event will the authors be held liable for any damages
arising from the use of this software.

Permission is granted to anyone to use this software for any purpose,
including commercial applications, and to alter it and redistribute it
freely, subject to the following restrictions:

1. The origin of this software must not be misrepresented; you must not
   claim that you wrote the original software. If you use this software
   in a product, an acknowledgment in the product documentation would be
   appreciated but is not required.
2. Altered source versions must be plainly marked as such, and must not be
   misrepresented as being the original software.
3. This notice may not be removed or altered from any source distribution.

<br>

## FFmpeg
Most files in FFmpeg are under the GNU Lesser General Public License version 2.1
or later (LGPL v2.1+). Read the file `COPYING.LGPLv2.1` for details. Some other
files have MIT/X11/BSD-style licenses. In combination the LGPL v2.1+ applies to
FFmpeg.

Some optional parts of FFmpeg are licensed under the GNU General Public License
version 2 or later (GPL v2+). See the file `COPYING.GPLv2` for details. None of
these parts are used by default, you have to explicitly pass `--enable-gpl` to
configure to activate them. In this case, FFmpeg's license changes to GPL v2+.

Specifically, the GPL parts of FFmpeg are:

- libpostproc
- optional x86 optimization in the files
    - `libavcodec/x86/flac_dsp_gpl.asm`
    - `libavcodec/x86/idct_mmx.c`
    - `libavfilter/x86/vf_removegrain.asm`
- the following building and testing tools
    - `compat/solaris/make_sunver.pl`
    - `doc/t2h.pm`
    - `doc/texi2pod.pl`
    - `libswresample/tests/swresample.c`
    - `tests/checkasm/*`
    - `tests/tiny_ssim.c`
- the following filters in libavfilter:
    - `signature_lookup.c`
    - `vf_blackframe.c`
    - `vf_boxblur.c`
    - `vf_colormatrix.c`
    - `vf_cover_rect.c`
    - `vf_cropdetect.c`
    - `vf_delogo.c`
    - `vf_eq.c`
    - `vf_find_rect.c`
    - `vf_fspp.c`
    - `vf_histeq.c`
    - `vf_hqdn3d.c`
    - `vf_kerndeint.c`
    - `vf_lensfun.c` (GPL version 3 or later)
    - `vf_mcdeint.c`
    - `vf_mpdecimate.c`
    - `vf_nnedi.c`
    - `vf_owdenoise.c`
    - `vf_perspective.c`
    - `vf_phase.c`
    - `vf_pp.c`
    - `vf_pp7.c`
    - `vf_pullup.c`
    - `vf_repeatfields.c`
    - `vf_sab.c`
    - `vf_signature.c`
    - `vf_smartblur.c`
    - `vf_spp.c`
    - `vf_stereo3d.c`
    - `vf_super2xsai.c`
    - `vf_tinterlace.c`
    - `vf_uspp.c`
    - `vf_vaguedenoiser.c`
    - `vsrc_mptestsrc.c`

Should you, for whatever reason, prefer to use version 3 of the (L)GPL, then
the configure parameter `--enable-version3` will activate this licensing option
for you. Read the file `COPYING.LGPLv3` or, if you have enabled GPL parts,
`COPYING.GPLv3` to learn the exact legal terms that apply in this case.

There are a handful of files under other licensing terms, namely:

* The files `libavcodec/jfdctfst.c`, `libavcodec/jfdctint_template.c` and
  `libavcodec/jrevdct.c` are taken from libjpeg, see the top of the files for
  licensing details. Specifically note that you must credit the IJG in the
  documentation accompanying your program if you only distribute executables.
  You must also indicate any changes including additions and deletions to
  those three files in the documentation.
* `tests/reference.pnm` is under the expat license.


## External libraries

FFmpeg can be combined with a number of external libraries, which sometimes
affect the licensing of binaries resulting from the combination.

### Compatible libraries

The following libraries are under GPL version 2:
- avisynth
- frei0r
- libcdio
- libdavs2
- librubberband
- libvidstab
- libx264
- libx265
- libxavs
- libxavs2
- libxvid

When combining them with FFmpeg, FFmpeg needs to be licensed as GPL as well by
passing `--enable-gpl` to configure.

The following libraries are under LGPL version 3:
- gmp
- libaribb24
- liblensfun

When combining them with FFmpeg, use the configure option `--enable-version3` to
upgrade FFmpeg to the LGPL v3.

The VMAF, mbedTLS, RK MPI, OpenCORE and VisualOn libraries are under the Apache License
2.0. That license is incompatible with the LGPL v2.1 and the GPL v2, but not with
version 3 of those licenses. So to combine these libraries with FFmpeg, the
license version needs to be upgraded by passing `--enable-version3` to configure.

The smbclient library is under the GPL v3, to combine it with FFmpeg,
the options `--enable-gpl` and `--enable-version3` have to be passed to
configure to upgrade FFmpeg to the GPL v3.

### Incompatible libraries

There are certain libraries you can combine with FFmpeg whose licenses are not
compatible with the GPL and/or the LGPL. If you wish to enable these
libraries, even in circumstances that their license may be incompatible, pass
`--enable-nonfree` to configure. This will cause the resulting binary to be
unredistributable.

The Fraunhofer FDK AAC and OpenSSL libraries are under licenses which are
incompatible with the GPLv2 and v3. To the best of our knowledge, they are
compatible with the LGPL.

<br>

## Font
- 본 저작물에서 사용한 제주고딕체는 공공누리 제1유형에 의거하여 [해당](https://www.jeju.go.kr/jeju/symbol/font/gothic.htm) 부분을 클릭하여 다운로드 받을 수 있음을 알려드립니다.

<br><br>

# 문법

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