BINARY_NAME=keyboard-backlight-daemon
OUTPUT_DIR=bin/

build:
	go build -o ${OUTPUT_DIR}${BINARY_NAME} main.go

run:
	go build -o ${OUTPUT_DIR}${BINARY_NAME} main.go
	./${OUTPUT_DIR}${BINARY_NAME}

clean:
	go clean
	rm ${OUTPUT_DIR}${BINARY_NAME}

deploy: build
	sudo systemctl stop keyboard-backlight-daemon
	sudo cp ${OUTPUT_DIR}${BINARY_NAME} /usr/bin/keyboard-backlight-daemon
	sudo chmod ug+x /usr/bin/keyboard-backlight-daemon
	sudo systemctl restart keyboard-backlight-daemon
