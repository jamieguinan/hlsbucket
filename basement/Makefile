PROJECTS?=$(HOME)/projects
VPATH=$(PROJECTS)/cti
INCLUDES=-I$(PROJECTS)/jsmn -I$(PROJECTS)/cti
LINK=-L$(PROJECTS)/jsmn -ljsmn

default: hlsbucket-c

%.o: %.c Makefile
	gcc -O $(INCLUDES) -c -Wall -Werror $< -o $@

hlsbucket-c: hlsbucket.o String.o Mem.o jsmn_extra.o File.o ArrayU8.o cti_utils.o bgprocess.o
	gcc -o $@ $^ $(LINK)

