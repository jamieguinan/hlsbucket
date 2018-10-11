VPATH=../cti
INCLUDES=-I../jsmn -I../cti
LINK=-L../jsmn -ljsmn

default: hlsbucket

hlsbucket: hlsbucket.go Makefile
	go build hlsbucket.go
	strip hlsbucket

%.o: %.c Makefile
	gcc -O $(INCLUDES) -c -Wall -Werror $< -o $@

hlsbucket-c: hlsbucket.o String.o Mem.o jsmn_extra.o File.o ArrayU8.o cti_utils.o bgprocess.o
	gcc -o $@ $^ $(LINK)

