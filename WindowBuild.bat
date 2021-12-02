@echo off

md RenG\core
md RenG\RenGLauncher
md RenG\game 
md RenG\Go

xcopy %~dp0Go\*.* %~dp0RenG\Go /e /h /k
xcopy %~dp0RenGLauncher\*.* %~dp0RenG\RenGLauncher /e /h /k
xcopy %~dp0game\*.* %~dp0RenG\game /e /h /k

copy %~dp0dll\*.dll %~dp0RenG\core

cd src
go build -o RenG.exe -ldflags -H=windowsgui main.go

cd ..
copy %~dp0src\RenG.exe %~dp0RenG\core
del %~dp0src\RenG.exe

go build -o RenGLauncher.exe -ldflags -H=windowsgui main.go
copy RenGLauncher.exe %~dp0RenGLauncher
del RenGLauncher.exe