SHELL = bash

win:
	common/windows/packfolder.exe pages resource.go -go
	cp common/windows/sciter.dll .
	go build -ldflags -H=windowsgui -o build/windows/silenda.exe
	mv ./sciter.dll build/windows

linux:
	common/linux/packfolder pages resource.go -go
	cp common/linux/*.so* .
	go build -o build/linux/silenda
	mv ./*.so* build/linux

darwin:
	common/darwin/packfolder pages resource.go -go
	cp common/darwin/*.dylib .
	go build -o build/darwin/silenda
	mv ./*.dylib build/darwin

clean:
	rm -rf resource.go
	rm -rf ./build/*/.silenda
	rm -rf ./build/*/*
