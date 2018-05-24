/* Catch Mpeg2TS packages over UDP, save as HLS compatible files. */

#include <stdio.h>
#include <stdint.h>
#include <inttypes.h>           /* PRiU64 */
#include <unistd.h>
#include <stdlib.h>             /* atoi */
#include <arpa/inet.h>
#include <sys/socket.h>


#include "../cti/MpegTS.h"

#define NAL_type(p) ((p)[4] & 31)

int main(int argc, char * argv[])
{
  uint64_t total = 0;
  char *e;
  if ((e = getenv("TOTAL_INIT"))) {
    total = atoll(e);
  }
  int udp_socket;
  struct sockaddr_in local = {};
  struct sockaddr_in remote = {};
  int port;
  
  if (argc != 2) {
    printf("Usage: %s <udp port number>\n", argv[0]);
    return 1;
  }
  port = atoi(argv[1]);

  udp_socket = socket(AF_INET, SOCK_DGRAM, IPPROTO_UDP);
  if (udp_socket == -1) {
    perror("socket");
    return 1;
  }

  local.sin_family = AF_INET;
  local.sin_addr.s_addr = htonl(INADDR_ANY);
  local.sin_port = htons(port);
  
  if (bind(udp_socket, (struct sockaddr *)&local, sizeof(local)) == -1) {
    perror("bind");
    return 1;
  }

  while (1) {
    unsigned int remote_len = sizeof(remote);  
    uint8_t buffer[32000];
    ssize_t n = recvfrom(udp_socket, buffer, sizeof(buffer), 0,
                     (struct sockaddr *) &remote, &remote_len);
    if (n <= 0) {
      break;
    }
    total += n;
    printf("%s: n=%zu total=%" PRIu64 " %" PRIu64 "MB %" PRIu64 "GB\n", __func__, n,
           total, total/(1024*1024), total/(1024*1024*1024));
  }
  return 0;
}
