# hlsbucket

hlsbucket is a program I wrote to receive video streams over UDP and store
as timestamped files. In my initial use case, the sender is a
[CTI](https://github.com/jamieguinan/cti) program running on a Raspberry Pi 2
with a Logitech webcam,
sending [H.264](https://en.wikipedia.org/wiki/H.264/MPEG-4_AVC)+[AAC](https://en.wikipedia.org/wiki/Advanced_Audio_Coding) in [Mpeg-TS](https://en.wikipedia.org/wiki/MPEG_transport_stream),
aiming to be compatible with [HLS](https://en.wikipedia.org/wiki/HTTP_Live_Streaming).


