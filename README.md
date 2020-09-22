# tftpd

For now, I'm learning TFTP protocol by testing it directly in my local tftpd server.

## Resources

- [RFC 1350](https://tools.ietf.org/html/rfc1350)
- [Wikipedia's article on Trivial File Transfer Protocol](https://en.wikipedia.org/wiki/Trivial_File_Transfer_Protocol)
- This _tcpdump_ command to inspect requests: `sudo tcpdump -i lo udp port 69 -vv -X`
- This _hexdump_ command to inspect the video file content in decimal form: `hexdump -n 10 -v -e '/1 "%03d\n"' ./video.avi`
- A [`tftpd-hpa`](https://packages.ubuntu.com/focal/net/tftpd-hpa) server instance on my Ubuntu development machine
