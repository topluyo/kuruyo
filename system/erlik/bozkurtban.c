#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <string.h>
#include <errno.h>
#include <unistd.h>
#include <fcntl.h>
#include <arpa/inet.h>
#include <sys/syscall.h>
#include <linux/bpf.h>
#include <time.h>

#define MAP_PATH "/sys/fs/bpf/bozkurt_blacklist"

struct IPKey {
    uint8_t Family;
    uint8_t Pad[3];
    union {
        uint32_t IPv4;
        uint8_t IPv6[16];
    } Addr;
};

struct blacklist_value {
    uint64_t expire_ns;
};

static int bpf_obj_get(const char *path)
{
    union bpf_attr attr;
    memset(&attr, 0, sizeof(attr));

    attr.pathname = (uint64_t)path;

    return syscall(__NR_bpf, BPF_OBJ_GET, &attr, sizeof(attr));
}

static int bpf_map_update_elem(int fd,
                               const void *key,
                               const void *value,
                               uint64_t flags)
{
    union bpf_attr attr;
    memset(&attr, 0, sizeof(attr));

    attr.map_fd = fd;
    attr.key = (uint64_t)key;
    attr.value = (uint64_t)value;
    attr.flags = flags;

    return syscall(__NR_bpf, BPF_MAP_UPDATE_ELEM, &attr, sizeof(attr));
}

static uint64_t monotonic_ns(void)
{
    struct timespec ts;

    clock_gettime(CLOCK_MONOTONIC, &ts);

    return (uint64_t)ts.tv_sec * 1000000000ULL + ts.tv_nsec;
}

static void usage(const char *prog)
{
    printf("Bozkurt Blacklist Tool\n");
    printf("\n");
    printf("Usage:\n");
    printf("    %s <ip> <second>\n", prog);
    printf("\n");
    printf("Examples:\n");
    printf("    %s 1.2.3.4 60\n", prog);
    printf("    %s 2001:db8::1 300\n", prog);
}

int main(int argc, char **argv)
{
    if (argc != 3) {
        usage(argv[0]);
        return 0;
    }

    const char *ip = argv[1];
    uint64_t seconds = strtoull(argv[2], NULL, 10);

    struct IPKey key;
    memset(&key, 0, sizeof(key));

    if (inet_pton(AF_INET, ip, &key.Addr.IPv4) == 1) {
        key.Family = AF_INET;
    } else if (inet_pton(AF_INET6, ip, key.Addr.IPv6) == 1) {
        key.Family = AF_INET6;
    } else {
        fprintf(stderr, "Invalid IP address: %s\n", ip);
        return 1;
    }

    struct blacklist_value value;
    value.expire_ns = monotonic_ns() + seconds * 1000000000ULL;

    int mapfd = bpf_obj_get(MAP_PATH);
    if (mapfd < 0) {
        fprintf(stderr, "Unable to open map %s: %s\n",
                MAP_PATH,
                strerror(errno));
        return 1;
    }

    if (bpf_map_update_elem(mapfd, &key, &value, BPF_ANY) != 0) {
        fprintf(stderr, "Failed to update map: %s\n",
                strerror(errno));
        close(mapfd);
        return 1;
    }

    printf("Added %s to blacklist for %llu seconds.\n",
           ip,
           (unsigned long long)seconds);

    close(mapfd);
    return 0;
}