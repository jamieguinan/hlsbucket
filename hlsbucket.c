/* Catch Mpeg2TS packets over UDP, save as .ts, generate .m3u8 for HLS */
#define _GNU_SOURCE
#include <stdio.h>
#include <stdint.h>
#include <inttypes.h>           /* PRiU64 */
#include <unistd.h>
#include <stdlib.h>             /* atoi */
#include <string.h>             /* memmem */
#include <arpa/inet.h>
#include <sys/socket.h>
#include "String.h"
#include "File.h"
#include "jsmn_extra.h"
#include "localptr.h"
#include "cti_utils.h"


#include "../cti/MpegTS.h"

#define CFGPATH "hlsbucket.json"

#define NAL_type(p) ((p)[4] & 31)

#define PKT_SIZE 188

typedef struct {
  const char * label;
  int numBytes;
  uint8_t bytes[];
} Target;
#define b(...) .numBytes = sizeof((uint8_t[]){ __VA_ARGS__ }), .bytes = { __VA_ARGS__ }
static Target nal7target = { .label = "naltype7", b(0x00,0x00,0x00,0x01,0x27)};

static FILE * fout = NULL;

void handle_packet(uint8_t pkt[PKT_SIZE], String * saveDir)
{
  int pid = MpegTS_PID(pkt);
  uint8_t * nal7pkt = NULL;
  if (pid == 258) {
    nal7pkt = memmem(pkt, PKT_SIZE, nal7target.bytes, nal7target.numBytes);
  }

  //printf("pid %d %s\n", pid, nal7pkt ? "NAL7" : "");

  /* If NAL7, start new segment. */
  if (nal7pkt) {
    if (fout) {
      fclose(fout);
    }
    double t;
    cti_getdoubletime(&t);
    localptr(String, outname) = String_sprintf("%s/out-%.3f.ts", s(saveDir), t);
    printf("new segment %s\n", s(outname));
    fout = fopen(s(outname), "wb");
  }

  if (fout) {
    if (fwrite(pkt, 188, 1, fout) != 1) {
      perror("fwrite"); fclose(fout); fout = NULL;
    }
  }
  
}

int main(int argc, char * argv[])
{
  uint64_t total = 0;
  int udp_socket;
  struct sockaddr_in local = {};
  struct sockaddr_in remote = {};
  String * saveDir = String_value_none();
  String * expireCommand = String_value_none();
  int hlsReceivePort = 0;
  int hlsRelayPort = 0;
  int i;
  
  char *e;
  
  if (argc != 1) {
    fprintf(stderr, "%s: specify options in %s\n", argv[0], CFGPATH);
    return 1;
  }
  
  {
    /* Load config */
    localptr(JsmnContext,jc) = JsmnContext_new();
    localptr(String, cfg) = File_load_text(S(CFGPATH));
    if (String_is_none(cfg)) {
      fprintf(stderr, "Could not load config file %s\n", CFGPATH);
      return 1;
    }
    JsmnContext_parse(jc, cfg);
    
    {
      struct  {
        const char * key;
        String ** value;
      } cfg_map[] = {
        {"saveDir", &saveDir }
        ,{"expireCommand", &expireCommand }
      };
      
      for (i=0; i < cti_table_size(cfg_map); i++) {
        *cfg_map[i].value = jsmn_lookup_string(jc, cfg_map[i].key);
        if (String_is_none(*cfg_map[i].value)) {
          fprintf(stderr, "%s not found in config\n", cfg_map[i].key);
          return 1;
        }
        else {
          printf("%s: \"%s\"\n", cfg_map[i].key, s(*cfg_map[i].value));
        }
      }
    }

    {
      struct  {
        const char * key;
        int * value;
      } cfg_map[] = {
        {"hlsReceivePort", &hlsReceivePort }
        ,{"hlsRelayPort", &hlsRelayPort }
      };
      
      for (i=0; i < cti_table_size(cfg_map); i++) {
        int rc = jsmn_lookup_int(jc, cfg_map[i].key, cfg_map[i].value);
        if (rc != 0) {
          fprintf(stderr, "%s not found in config\n", cfg_map[i].key);        
          return 1;
        }
        else {
          printf("%s: %d\n", cfg_map[i].key, *cfg_map[i].value); 
        }
      }
    }

  }


  udp_socket = socket(AF_INET, SOCK_DGRAM, IPPROTO_UDP);
  if (udp_socket == -1) {
    perror("socket");
    return 1;
  }

  local.sin_family = AF_INET;
  local.sin_addr.s_addr = htonl(INADDR_ANY);
  local.sin_port = htons(hlsReceivePort);
  
  if (bind(udp_socket, (struct sockaddr *)&local, sizeof(local)) == -1) {
    perror("bind");
    return 1;
  }

  if ((e = getenv("TOTAL_INIT"))) {
    total = atoll(e);
  }

  while (1) {
    unsigned int remote_len = sizeof(remote);  
    uint8_t buffer[32000];
    ssize_t n = recvfrom(udp_socket, buffer, sizeof(buffer), 0,
                     (struct sockaddr *) &remote, &remote_len);
    if (n <= 0) {
      perror("recvfrom");
      sleep(1);
      continue;
    }

    total += n;
    //printf("%s: n=%zu total=%" PRIu64 " %" PRIu64 "MB %" PRIu64 "GB\n", __func__, n,  total, total/(1024*1024), total/(1024*1024*1024));

    if (n % PKT_SIZE != 0) {
      fprintf(stderr, "datagram is not a multiple of %d bytes\n", PKT_SIZE);
      sleep(1);      
      continue;
    }
    
    for (i=0; i < n; i+=PKT_SIZE) {
      handle_packet(&buffer[i], saveDir);
    }
  }
  return 0;
}
