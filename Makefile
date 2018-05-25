VPATH=../cti
INCLUDES=-I../jsmn -I../cti
LINK=-L../jsmn -ljsmn

%.o: %.c Makefile
	gcc -O $(INCLUDES) -c -Wall -Werror $< -o $@

hlsbucket: hlsbucket.o String.o Mem.o dpf.o jsmn_extra.o File.o ArrayU8.o
	gcc -o $@ $^ $(LINK)

