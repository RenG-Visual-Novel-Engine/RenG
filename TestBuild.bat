@echo off

md Blue\core
md Blue\game

xcopy %~dp0test\*.* %~dp0Blue\game /e /h /k

copy %~dp0dll\*.dll %~dp0Blue\core

cd src
go build -o RenG.exe -ldflags -H=windowsgui main.go

cd ..
copy %~dp0src\RenG.exe %~dp0Blue\core
del %~dp0src\RenG.exe

go build -o Blue.exe -ldflags -H=windowsgui main.go
copy Blue.exe %~dp0Blue
del Blue.exe