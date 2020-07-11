# hlsbucket

## Overview and quick start

hlsbucket receives H264+AAC MpegTS video streams over UDP and stores
them in timestamped folders and files, and dynamically generates HLS
stream info and index content. It serves these files over HTTP on a
configurable port.

Assuming a Linux system with Go 1.1x, ffmpeg and ffplay, and a UVC compatible web cam,

     # terminal 1
    go build
    ./hlsbucket
    
    # terminal 2
    ffmpeg -f v4l2 -i /dev/video0 -f alsa -c:v libx264 -g 90 -f mpegts -flush_packets 0 udp://localhost:6679?pkt_size=131

    # terminal 3
    ffplay http://localhost:8004/play

If that all works, try browsing to http://your.local.ip.addr:8004/play on a mobile browser.

## In more detail

With the included example `hlsbucket.json` configuration,

    {
        "SaveDir":"ts"
        ,"HlsReceivePort":6679
        ,"HlsListenPort":8004
        ,"ExpireTime":"30m"
        ,"StartCode":"00.00.00.01.68"
    }

hlsbucket will generate this stream info content at `http://localhost:8004/play`

    #EXTM3U
    #EXT-X-STREAM-INF:PROGRAM-ID=1, BANDWIDTH=200000
    live_index.m3u8

this index at `http://localhost:8004/live_index.m3u8`

    #EXTM3U
    #EXT-X-TARGETDURATION:3
    #EXT-X-VERSION:3
    #EXT-X-MEDIA-SEQUENCE:21
    #EXTINF:3, no desc
    ts/2020/07/06/15/1594049947.973.ts
    #EXTINF:3, no desc
    ts/2020/07/06/15/1594049950.952.ts
    #EXTINF:3, no desc
    ts/2020/07/06/15/1594049953.969.ts

And these file urls,

    http://localhost:8004/ts/2020/07/06/15/1594049947.973.ts
    http://localhost:8004/ts/2020/07/06/15/1594049950.952.ts
    http://localhost:8004/ts/2020/07/06/15/1594049953.969.ts
    ...

The actual files are stored at,

    ts/2020/07/06/15/1594049947.973.ts
    ts/2020/07/06/15/1594049950.952.ts
    ts/2020/07/06/15/1594049953.969.ts
    ...

The files will pile up over time, so "ExpireTime" is set to 30 minutes
(time must be formatted according to
`https://golang.org/pkg/time/#ParseDuration`), and a background thread
will clean up files older than 30 minutes.

All this assumes,

  * There is a program or device sending an MpegTS stream over the network to UDP port 6679
    (ffmpeg in the above example).
  * The stream contains packets with matching start code bytes at regular intervals.

## Limitations

  * Only supports a single stream, so the "Hierarchical" part of
    HLS isn't really there.
  * There is always a delay of several seconds between audio+video
    capture and HLS playback. This is due to the way HLS serves
    complete files, which are typically several seconds long, and
    only appear in the index when complete.

## Basic operation

At the core of hlsbucket is a for loop iterates over the received UDP
data and passes 188-byte chunks to `handlePacket()`, which looks for a
matching byte pattern and starts a new `.ts` segment file whenever it
finds one.

## ffmpeg sender example

On a Linux system with a UVC camera, this should work,

    ffmpeg -f v4l2 -i /dev/video0 -f alsa -c:v libx264 -g 90 -f mpegts -flush_packets 0 udp://localhost:6679?pkt_size=1316

## CTI sender

I originally wrote hlsbucket to work with a
[CTI](https://github.com/jamieguinan/cti/blob/master/README.md)
program, which [muxes and
segments](https://github.com/jamieguinan/cti/blob/master/MpegTSMux.c)
[H.264](https://github.com/jamieguinan/cti/blob/master/RPiH264Enc.c)
and [AAC](https://github.com/jamieguinan/cti/blob/master/AAC.c) data
into 188-byte TS packets.

Some network overhead can be saved by aggregating a few packets before
transmission over UDP.  On my home network, `ifconfig` says the
[MTU](https://en.wikipedia.org/wiki/Maximum_transmission_unit) is
1500, but after some experimentation with tcpdump,

    13:45:40.232471 IP 192.168.1.16 > 192.168.1.14: ip-proto-17
    13:45:40.232497 IP 192.168.1.16.48440 > 192.168.1.14.6679: UDP, bad length 1880 > 1472
    13:45:40.232499 IP 192.168.1.16 > 192.168.1.14: ip-proto-17
    13:45:40.232506 IP 192.168.1.16.48440 > 192.168.1.14.6679: UDP, bad length 1880 > 1472

I used 1472 as an upper limit, and [this stackoverflow post](https://stackoverflow.com/questions/14993000/the-most-reliable-and-efficient-udp-packet-size) confirms that number.

In the sender cmd file I configure the [UDPTransmit](https://github.com/jamieguinan/cti/blob/master/UDPTransmit.c) instance with,

    config ut buffer_level 1316

for 7x 188-byte packets.


## Other notes

Since CTI is not popular and might be difficult to build and use, I
added the ffmpeg example for users that find the project on Github.

In the `basement/` folder is the original C version of hlsbucket. The
Go version was one my first attempts at programming with Go. It took
2/3 as many lines of code, and I was able to use Go standard libraries
for everything I needed.

## Alternative HLS solutions,

  * Apple's
    [https://developer.apple.com/documentation/http_live_streaming](HTTP
    Live Streaming) page notes several frameworks that support HLS
    generation.
  * ffmpeg supports an [hls muxer](https://ffmpeg.org/ffmpeg-formats.html#hls-2)
  * Many others [on github](https://github.com/search?q=hls+server)
