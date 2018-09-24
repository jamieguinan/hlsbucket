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
#include <time.h>               /* gmtime_r */
#include <sys/time.h>           /* gettimeofday */
#include <sys/stat.h>           /* mkdir */
#include <sys/types.h>          /* mkdir */

#include "String.h"
#include "File.h"
#include "jsmn_extra.h"
#include "localptr.h"
#include "cti_utils.h"
#include "bgprocess.h"


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

static int checkdirs(String * root, String * path)
{
  fprintf(stderr, "%s: %s / %s\n", __func__, s(root), s(path));
  struct stat ds;
  localptr(String, fullpath) = String_sprintf("%s/%s", s(root), s(path));
  if (stat(s(fullpath), &ds) == 0) {
    return 0;
  }

  localptr(String, dname) = String_dirname(path);
  if (String_eq(dname, path)) {
    /* Can't reduce any further */
    fprintf(stderr, "can't reduce any further, trying mkdir");
    localptr(String, dpath) = String_sprintf("%s/%s", s(root), s(dname));
    return mkdir(s(dpath), 0744);
  }
  else {
    int rc1 = checkdirs(root, dname);
    fprintf(stderr, "recurse returned %d\n", rc1);
    int rc2 = mkdir(s(fullpath), 0744);
    fprintf(stderr, "mkdir(%s) returned %d\n", s(fullpath), rc2);
    if ((rc1 == 0) && (rc2 == 0)) {
      return 0;
    }
    else {
      return 1;
    }
  }
}

static struct {
  String * expireCommand;
  FILE * fout;
} private = {
  .expireCommand = NULL
  ,.fout = NULL
};

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
    if (private.fout) {
      fclose(private.fout); private.fout = NULL;
    }
    struct timeval tv;
    struct tm gt;
    double dt;
    gettimeofday(&tv, NULL);
    dt = (tv.tv_sec + (tv.tv_usec/1000000.0));
    gmtime_r(&tv.tv_sec, &gt);
    localptr(String, midpath) = String_sprintf("%04d/%02d/%02d/%02d"
                                               ,gt.tm_year+1900
                                               ,gt.tm_mon+1
                                               ,gt.tm_mday
                                               ,gt.tm_hour
                                               );

    if (checkdirs(saveDir, midpath) != 0) {
      return;
    }
    localptr(String, outname) = String_sprintf("%s/%s/%.3f.ts", s(saveDir), s(midpath), dt);
    printf("new segment %s\n", s(outname));
    private.fout = fopen(s(outname), "wb");
    if (private.fout && private.expireCommand) {
      char * args[] = { s(private.expireCommand), s(outname), "10d", NULL };
      bgreap_one();
      bgstartv(args, NULL);
    }
  }

  if (private.fout) {
    if (fwrite(pkt, 188, 1, private.fout) != 1) {
      perror("fwrite"); fclose(private.fout); private.fout = NULL;
    }
  }

}

int main(int argc, char * argv[])
{
  uint64_t total = 0;
  int receive_socket;
  int relay_socket;
  struct sockaddr_in local = {};
  struct sockaddr_in remote = {};
  struct sockaddr_in relay_remote = {};
  String * saveDir = String_value_none();
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
        ,{"expireCommand", &private.expireCommand }
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

  {
    /* Receive socket setup */
    receive_socket = socket(AF_INET, SOCK_DGRAM, IPPROTO_UDP);
    if (receive_socket == -1) {
      perror("socket");
      return 1;
    }

    local.sin_family = AF_INET;
    local.sin_addr.s_addr = htonl(INADDR_ANY);
    local.sin_port = htons(hlsReceivePort);

    if (bind(receive_socket, (struct sockaddr *)&local, sizeof(local)) == -1) {
      perror("bind (receive socket)");
      return 1;
    }
  }

  {
    /* Relay socket setup */
    relay_socket = socket(AF_INET, SOCK_DGRAM, IPPROTO_UDP);
    if (relay_socket == -1) {
      perror("socket");
      return 1;
    }

    local.sin_family = AF_INET;
    local.sin_addr.s_addr = htonl(INADDR_ANY);
    local.sin_port = htons(hlsRelayPort);

    if (bind(relay_socket, (struct sockaddr *)&local, sizeof(local)) == -1) {
      perror("bind (receive socket)");
      return 1;
    }
  }

  /* Prepare and loop. */
  if ((e = getenv("TOTAL_INIT"))) {
    total = atoll(e);
  }

  while (1) {
    /* Receive socket */
    unsigned int remote_len = sizeof(remote);
    uint8_t buffer[32000];
    ssize_t n = recvfrom(receive_socket, buffer, sizeof(buffer), 0,
                         (struct sockaddr *) &remote, &remote_len);
    if (n <= 0) {
      perror("recvfrom");
      sleep(1);
      continue;
    }

    total += n;
    // printf("%s: n=%zu total=%" PRIu64 " %" PRIu64 "MB %" PRIu64 "GB\n", __func__, n,  total, total/(1024*1024), total/(1024*1024*1024));

    if (n % PKT_SIZE != 0) {
      fprintf(stderr, "datagram is not a multiple of %d bytes\n", PKT_SIZE);
      sleep(1);
      continue;
    }

    for (i=0; i < n; i+=PKT_SIZE) {
      handle_packet(&buffer[i], saveDir);
    }


    /* Relay socket */
    unsigned int relay_remote_len = sizeof(relay_remote);
    uint8_t request[32000];
    ssize_t reqn = recvfrom(relay_socket, request, sizeof(request), MSG_DONTWAIT,
                            (struct sockaddr *) &relay_remote, &relay_remote_len);
    if (reqn > 0) {
      /* Set relay target. */
      printf("<- got relay request\n");
    }

    if (relay_remote.sin_port) {
      ssize_t rn = sendto(relay_socket, buffer, n, 0,
                          (struct sockaddr *) &relay_remote, relay_remote_len);
      if (rn < 0) {
        perror("sendto");
      }
    }
  }

  return 0;
}
