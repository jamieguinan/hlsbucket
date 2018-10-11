# hlsbucket

hlsbucket is a program I wrote to receive video streams over UDP and
store them in timestamped folders and files. It also allows a single
relay connection to another receiver.

In my initial use case, the sender is a
[CTI](https://github.com/jamieguinan/cti) program running on a Raspberry Pi 2
with a Logitech webcam,
sending [H.264](https://en.wikipedia.org/wiki/H.264/MPEG-4_AVC)+[AAC](https://en.wikipedia.org/wiki/Advanced_Audio_Coding) in [Mpeg-TS](https://en.wikipedia.org/wiki/MPEG_transport_stream),
aiming to be compatible with [HLS](https://en.wikipedia.org/wiki/HTTP_Live_Streaming).

I initially wrote hlsbucket in C, but decided to port it to Go, and I expect to
maintain only the Go version going forward. The
[first Go version](https://github.com/jamieguinan/hlsbucket/blob/aedbb8232692ab92f78bd0e2b0faab5e8e4c1986/hlsbucket.go) to reach feature parity with [the C version](https://github.com/jamieguinan/hlsbucket/blob/aedbb8232692ab92f78bd0e2b0faab5e8e4c1986/hlsbucket.c)
was about 2/3 as many lines of code. I was able to use Go standard libraries instead of requiring CTI modules and [Jsmn](https://github.com/zserge/jsmn).

