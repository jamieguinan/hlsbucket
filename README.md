# hlsbucket

hlsbucket is a program I wrote to receive video streams over UDP and
store them in timestamped folders and files. It also allows a single
relay connection to another receiver.

In my initial use case, the sender is a
[CTI](https://github.com/jamieguinan/cti) program running on a Raspberry Pi 2
with a Logitech webcam,
sending [H.264](https://en.wikipedia.org/wiki/H.264/MPEG-4_AVC)+[AAC](https://en.wikipedia.org/wiki/Advanced_Audio_Coding) in [Mpeg-TS](https://en.wikipedia.org/wiki/MPEG_transport_stream),
aiming to be compatible with [HLS](https://en.wikipedia.org/wiki/HTTP_Live_Streaming).

I initially wrote hlsbucket in C, but decided to port it to [Go](https://golang.org/),
and I expect to maintain only the Go version going forward. The
[first Go version](https://github.com/jamieguinan/hlsbucket/blob/aedbb8232692ab92f78bd0e2b0faab5e8e4c1986/hlsbucket.go) to reach feature parity with [the C version](https://github.com/jamieguinan/hlsbucket/blob/aedbb8232692ab92f78bd0e2b0faab5e8e4c1986/hlsbucket.c)
was about 2/3 as many lines of code, and I was able to use Go standard libraries instead of requiring CTI modules.

## How it works, sender

The sender code isn't part of this project (it is a [CTI script](https://github.com/jamieguinan/cti/blob/master/README.md)) but its behavior drives hlsbucket's implementation.

The sender [muxes and segments](https://github.com/jamieguinan/cti/blob/master/MpegTSMux.c)
[H.264](https://github.com/jamieguinan/cti/blob/master/RPiH264Enc.c)
and [AAC](https://github.com/jamieguinan/cti/blob/master/AAC.c)
data into 188-byte TS packets. I can save some network overhead by
aggregating a few packets before transmission over UDP.
`ifconfig` says the [MTU](https://en.wikipedia.org/wiki/Maximum_transmission_unit)
on my home network is 1500, but after some experimentation with tcpdump,

    13:45:40.232471 IP 192.168.1.16 > 192.168.1.14: ip-proto-17
    13:45:40.232497 IP 192.168.1.16.48440 > 192.168.1.14.6679: UDP, bad length 1880 > 1472
    13:45:40.232499 IP 192.168.1.16 > 192.168.1.14: ip-proto-17
    13:45:40.232506 IP 192.168.1.16.48440 > 192.168.1.14.6679: UDP, bad length 1880 > 1472

I used 1472 as an upper limit, and [this stackoverflow post](https://stackoverflow.com/questions/14993000/the-most-reliable-and-efficient-udp-packet-size) confirms that number.

In the sender cmd file I configure the [UDPTransmit](https://github.com/jamieguinan/cti/blob/master/UDPTransmit.c) instance with,

    config ut buffer_level 1316

for 7x 188 packets.

## How it works, receiver

In hlsbucket, a for loop iterates over the received UDP data and
passes 188-byte chunks to `handlePacket()`, which looks for
[NAL](https://en.wikipedia.org/wiki/Network_Abstraction_Layer) type 7
packets and starts a new segment whenever it finds one.


## Adding a web server

After the initial C-to-Go port with file archiving, I wanted to be able to generate HLS index files dynamically, so that I could serve views to web browsers. [MpegTSMux.c](https://github.com/jamieguinan/cti/blob/master/MpegTSMux.c) has some of this capability built-in, but is has some limitations,

  * It writes files rather than dynamically generating them.
  * It only tracks the "live" stream.
  * It doesn't allow for an idea I had some time ago: predicting the next segment,
    and sending (relaying) it as it is received. I think this could tighten up live
    playback delay from a few seconds to less than a second.

[ I have already written a simple web service application (instaskunk), so I hope it will be easy to add the functionality I want to hlsbucket. ]
