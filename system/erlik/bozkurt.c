#include <linux/bpf.h>        // eBPF verileri ve sabitleri için temel kütüphane
#include <bpf/bpf_helpers.h>  // bpf_map_lookup_elem, bpf_ktime_get_ns vb. eBPF yardımcı fonksiyonları
#include <linux/if_ether.h>   // Ethernet başlık yapıları (ethhdr, ETH_P_IP vb.)
#include <linux/ip.h>         // IPv4 başlık yapısı (iphdr)
#include <linux/ipv6.h>       // IPv6 başlık yapısı (ipv6hdr)
#include <linux/tcp.h>        // TCP başlığı
#include <linux/udp.h>        // UDP başlığı
#include <linux/in.h>         // IPPROTO_TCP, vb.



//@IPv4CIDR
struct IPv4CIDR{
	__u32 prefixlen;
    union {
		__u8 bytes[4];
		__be32 IPv4;
	} addr;
};
//@IPv6CIDR
struct IPv6CIDR{
	__u32 prefixlen;
	union {
		__u8 bytes[16];
		__be64 IPv6[2];
	} addr;
};

struct IPKey {
	union {
		__u8 Bytes[16];
		__be32 IPv4;
		__be64 IPv6[2];
	} Addr;
};

struct TCPInfo {
    __be16 SrcPort;
    __be16 DstPort;
    __u8 Flags;
    __u8 DoL;
};

struct PacketInfo {
	struct IPKey IP;
	__be16 InPort;
    __be16 OtPort;
	__u8 Protocol;
	__u8 Pad;
	__u32 Length;
	__u8 IsFragmented;
	__u8 Family;
	struct TCPInfo TCP;
	__u32 Info;
};	


static __always_inline int ParsePacket(
    void *data,
    void *data_end,
    struct PacketInfo *packet)
{
    struct ethhdr *eth = data;

    /* Ethernet header */
    if ((void *)(eth + 1) > data_end)
        return 0;

    /* Initialize output fields */
    packet->Length      = (__u32)(data_end - data);
    packet->InPort      = 0;
    packet->OtPort      = 0;
    packet->Protocol    = 0;
    packet->Pad         = 0;
    packet->IsFragmented = 0;
    packet->Family      = 0;
    packet->Info        = 0;

    packet->TCP.SrcPort = 0;
    packet->TCP.DstPort = 0;
    packet->TCP.Flags   = 0;
    packet->TCP.DoL     = 0;

    switch (eth->h_proto) {

    /* =========================================================
     * IPv4
     * ========================================================= */
    case __constant_htons(ETH_P_IP): {
        struct iphdr *ip = (void *)(eth + 1);

        if ((void *)(ip + 1) > data_end)
            return 0;

        /*
         * IPv4 IHL is the number of 32-bit words.
         * Minimum legal IPv4 header is 20 bytes.
         */
        if (ip->ihl < 5)
            return 0;

        int iph_len = ip->ihl * 4;

        if ((void *)ip + iph_len > data_end)
            return 0;

        packet->Family   = 2; /* AF_INET */
        packet->IP.Addr.IPv4 = ip->saddr;
        packet->Protocol = ip->protocol;

        /*
         * Evil Bit / Reserved Fragment Flag.
         *
         * frag_off:
         *   bit 15 = reserved
         *   bits 13-14 = DF/MF
         *   bits 0-12 = fragment offset
         */
        if (ip->frag_off & __constant_htons(0x8000))
            return 0;

        /*
         * Fragmentation detection:
         *
         * MF flag OR non-zero fragment offset.
         *
         * 0x3FFF covers MF + fragment offset.
         */
        if (ip->frag_off & __constant_htons(0x3FFF))
            packet->IsFragmented = 1;
        else
            packet->IsFragmented = 0;

        /*
         * For fragmented IPv4 packets, the transport header
         * may not be present. Do not attempt TCP/UDP parsing.
         */
        if (packet->IsFragmented)
            return 1;

        /* Transport header starts after IPv4 header */
        void *transport = (void *)ip + iph_len;

        /* -----------------------------------------------------
         * TCP
         * ----------------------------------------------------- */
        if (packet->Protocol == IPPROTO_TCP) {
            struct tcphdr *tcp = transport;

            if ((void *)(tcp + 1) > data_end)
                return 1;

            packet->InPort = tcp->dest;
            packet->OtPort = tcp->source;
            packet->TCP.SrcPort = tcp->source;
            packet->TCP.DstPort = tcp->dest;

            /*
             * TCP flags are at byte 13.
             * Data offset is contained in the upper nibble
             * of byte 12.
             */
            __u8 *tcp_base = (__u8 *)tcp;

            if ((void *)(tcp_base + 14) <= data_end) {
                packet->TCP.DoL   = tcp_base[12];
                packet->TCP.Flags = tcp_base[13];
            }
        }

        /* -----------------------------------------------------
         * UDP
         * ----------------------------------------------------- */
        else if (packet->Protocol == IPPROTO_UDP) {
            struct udphdr *udp = transport;

            if ((void *)(udp + 1) > data_end)
                return 1;

            packet->InPort = udp->dest;
            packet->OtPort = udp->source;
        }

        return 1;
    }

    /* =========================================================
     * IPv6
     * ========================================================= */
    case __constant_htons(ETH_P_IPV6): {
        struct ipv6hdr *ip6 = (void *)(eth + 1);

        if ((void *)(ip6 + 1) > data_end)
            return 0;

        packet->Family = 10; /* AF_INET6 */

        __builtin_memcpy(
            packet->IP.Addr.IPv6,
            &ip6->saddr,
            sizeof(packet->IP.Addr.IPv6)
        );

        packet->Protocol = ip6->nexthdr;

        /*
         * Basic IPv6 Fragment Header detection.
         *
         * IPPROTO_FRAGMENT = 44
         */
        if (packet->Protocol == IPPROTO_FRAGMENT) {
            packet->IsFragmented = 1;
            return 1;
        }

        /*
         * Current implementation handles TCP/UDP directly
         * after the IPv6 base header.
         *
         * Extension-header walking can be added separately
         * if required.
         */
        void *transport = (void *)(ip6 + 1);

        /* -----------------------------------------------------
         * TCP
         * ----------------------------------------------------- */
        if (packet->Protocol == IPPROTO_TCP) {
            struct tcphdr *tcp = transport;

            if ((void *)(tcp + 1) > data_end)
                return 1;

            packet->InPort = tcp->dest;
            packet->OtPort = tcp->source;

            packet->TCP.SrcPort = tcp->source;
            packet->TCP.DstPort = tcp->dest;

            __u8 *tcp_base = (__u8 *)tcp;

            if ((void *)(tcp_base + 14) <= data_end) {
                packet->TCP.DoL   = tcp_base[12];
                packet->TCP.Flags = tcp_base[13];
            }
        }

        /* -----------------------------------------------------
         * UDP
         * ----------------------------------------------------- */
        else if (packet->Protocol == IPPROTO_UDP) {
            struct udphdr *udp = transport;

            if ((void *)(udp + 1) > data_end)
                return 1;

            packet->InPort = udp->dest;
            packet->OtPort = udp->source;
        }

        return 1;
    }

    /* =========================================================
     * Unsupported EtherType
     * ========================================================= */
    default:
        return 0;
    }
}


//@SYN_STATS
struct {
    __uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
    __uint(max_entries, 1);
    __type(key, __u32);
    __type(value, __u64);
} SYN_STATS SEC(".maps");


//@ ParsePacketByCTX
static __always_inline int ParsePacketByCTX(struct xdp_md *ctx, struct PacketInfo *packet){
    void *data = (void *)(long)ctx->data;
    void *data_end = (void *)(long)ctx->data_end;

    return ParsePacket(data, data_end, packet);
}

//@ CheckTCPFlags (Anomali Tespiti)
static __always_inline int CheckTCPFlags(struct TCPInfo *tcp) {
    __u8 flags = tcp->Flags;
    
    // Bayrak Tanımları
    __u8 FIN = 0x01;
    __u8 SYN = 0x02;
    __u8 RST = 0x04;
    __u8 PSH = 0x08;
    __u8 ACK = 0x10;
    __u8 URG = 0x20;

    // 1. SYN ve FIN aynı anda aktifse (SYN-FIN Taraması)
    if ((flags & SYN) && (flags & FIN)) return 0;
    
    // 2. SYN ve RST aynı anda aktifse
    if ((flags & SYN) && (flags & RST)) return 0;
    
    // 3. Hiçbir bayrak yoksa (Null Scan)
    if (flags == 0) return 0;
    
    // 4. FIN, PSH, URG aynı anda aktifse (Xmas Scan)
    if ((flags & FIN) && (flags & PSH) && (flags & URG)) return 0;
    
    // 5. FIN var ama ACK yoksa (FIN Scan)
    if ((flags & FIN) && !(flags & ACK)) return 0;

    return 1; // Geçerli Paket
}

// SYN Flood Koruması için LRU Hash Map
struct syn_tracker_value {
    __u64 count;
    __u64 first_seen;
};





//@ Sync
struct SyncObject {
    __u32 period;
    __u32 time;
    __u32 mode;
};
volatile struct SyncObject SYNC SEC(".bss");;




//# 138:MAP IP_DURATION white
struct {
	/* Harita türü   */     __uint(type,         BPF_MAP_TYPE_HASH);        
	/* Maksimum Data */     __uint(max_entries,  1000000);         
	/* Anahtar       */     __type(key,          struct IPKey);              
	/* Değer         */     __type(value,        __u64);  
} white SEC(".maps");

//# 139:MAP IP_DURATION block
struct {
	/* Harita türü   */     __uint(type,         BPF_MAP_TYPE_HASH);        
	/* Maksimum Data */     __uint(max_entries,  1000000);         
	/* Anahtar       */     __type(key,          struct IPKey);              
	/* Değer         */     __type(value,        __u64);  
} block SEC(".maps");

//# 141:MAP IP dns
struct {
	/* Harita türü   */     __uint(type,         BPF_MAP_TYPE_HASH);        
	/* Maksimum Data */     __uint(max_entries,  1000000);         
	/* Anahtar       */     __type(key,          struct IPKey);              
	/* Değer         */     __type(value,        __u8);  
} dns SEC(".maps");

//# 153:MAP INPORT enableport
struct {
	/* Harita türü   */     __uint(type,         BPF_MAP_TYPE_ARRAY);        
	/* Maksimum Data */     __uint(max_entries,  65536);         
	/* Anahtar       */     __type(key,          __u32);
	/* Değer         */     __type(value,        __u8);  
} enableport SEC(".maps");

//# 158:MAP IP_DURATION fragmented
struct {
	/* Harita türü   */     __uint(type,         BPF_MAP_TYPE_HASH);        
	/* Maksimum Data */     __uint(max_entries,  1000000);         
	/* Anahtar       */     __type(key,          struct IPKey);              
	/* Değer         */     __type(value,        __u64);  
} fragmented SEC(".maps");

//# 173:MAP IP_DURATION synblock
struct {
	/* Harita türü   */     __uint(type,         BPF_MAP_TYPE_HASH);        
	/* Maksimum Data */     __uint(max_entries,  1000000);         
	/* Anahtar       */     __type(key,          struct IPKey);              
	/* Değer         */     __type(value,        __u64);  
} synblock SEC(".maps");




struct {
    __uint(type, BPF_MAP_TYPE_LRU_HASH);
    __uint(max_entries, 100000);
    __type(key, struct IPKey);
    __type(value, struct syn_tracker_value);
} syn_rate_limit SEC(".maps");

//! FOR TEST
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 400000); 
    __type(key, struct IPKey);   
    __type(value, __u8);  
} UNIQ_IP SEC(".maps");
const __u8 ONE  = 1;
static __always_inline __u32 Uniq_IP(const struct PacketInfo *pkt) {
    bpf_map_update_elem(&UNIQ_IP,&pkt->IP,&ONE,BPF_ANY);
}

// @HLL_TABLE 4096 kovalı bir HLL haritası (Sadece 4 KB yer kaplar!)
struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __uint(max_entries, 16384); 
    __type(key, __u32);   // Kova indeksi (0 - 4095)
    __type(value, __u8);  // O kovadaki maksimum ardışık sıfır sayısı
} HLL_TABLE SEC(".maps");

//@ Hash
static __always_inline __u32 Hash(const struct PacketInfo *pkt, __u32 seed) {
    // Knuth's Golden Ratio Constant (2^32 / Golden Ratio)
    const __u32 K = 0x9E3779B9U;
    
    // Seed, IPv4/IPv6 uzunluk ve protokol ön yüklemesi
    __u32 h = seed ^ (pkt->Family * K);

    if (pkt->Family == 2) { // AF_INET
        // IPv4: Tek çarpma ve karıştırma
        h ^= pkt->IP.Addr.IPv4;
        h *= K;
    } else { // AF_INET6
        // IPv6: Unrolled 64-bit okumalar ile ALU cycle'ı yarıya indirme
        const __u64 *d = (const __u64 *)pkt->IP.Addr.Bytes;
        
        // 128-bit IPv6 adresini 2x 64-bit olarak katlayıp karıştırıyoruz
        __u32 low  = (__u32)d[0] ^ (__u32)(d[0] >> 32);
        __u32 high = (__u32)d[1] ^ (__u32)(d[1] >> 32);

        h ^= low;
        h *= K;
        h ^= high;
        h *= K;
    }

    // Final Avalanche Step (Daha az shift ile Knuth çığ etkisi)
    h ^= h >> 16;
    return h * K;
}

//@ Uniq
static __always_inline void Uniq(struct PacketInfo *pkt){
    __u32 hash = Hash(pkt, 4);

    // 16384 kova için ilk 14 bit kova indeksi (16383 maskesi: 0x3FFF)
    __u32 bucket = hash & 0x3FFF;       
    // Geriye kalan 18 bit remain için (32 - 14 = 18)
    __u32 remain = (hash >> 14) & 0x3FFFF; 

    // 18 bitlik alanda __builtin_clz doğru çalışması için 19. pozisyona bekçi bit ekliyoruz
    __u32 val = remain | 0x80000; 
    __u8 zeros = (__u8)(32 - __builtin_clz(val) - 13); 
    // 32 - clz ile en yüksek bitin pozisyonunu bulup 13 çıkararak 18-bit HLL derecesine normalleştiriyoruz.

    __u8 *current_max = bpf_map_lookup_elem(&HLL_TABLE, &bucket);
    if (current_max && zeros > *current_max) {
        *current_max = zeros; 
    }
}

//@ Stat
struct Report {
    __u32 pass;
    __u32 drop;
    __u64 byte;
    __u32 uniq;
};
struct {
    __uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
    __uint(max_entries, 1440);
    __type(key, __u32);
    __type(value, struct Report);
} REPORT SEC(".maps");

static __always_inline void Stat(struct PacketInfo *pkt, __u8 pass){
    if (SYNC.period >= 1440) return;
    struct Report *r = bpf_map_lookup_elem(&REPORT, &SYNC.period);
    if (!r) return;
    r->byte += pkt->Length;
    if (pass){
        r->pass++;
    }else{
        r->drop++;
    }
}


//@ Log
struct {
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, 4 * 1024 * 1024); 
} LOGS SEC(".maps");

static __always_inline void Log(struct PacketInfo *packet, __u32 info, __u8 pass){
    struct PacketInfo *e;
    e = bpf_ringbuf_reserve(&LOGS, sizeof(*e), 0);
    if (!e) return;
    packet->Info = info;
    __builtin_memcpy(e, packet, sizeof(*e));
    bpf_ringbuf_submit(e, 0);
}

//@ Bozkurt: XDP (eXpress Data Path) Programının Giriş Noktası
SEC("xdp")
int Bozkurt(struct xdp_md *ctx) {
    void *data_end = (void *)(long)ctx->data_end;
    void *data     = (void *)(long)ctx->data;
    struct ethhdr *eth = data;

    if ((void *)(eth + 1) > data_end)
        return XDP_PASS;

    /* 
        KRONİK SORUNUN ÇÖZÜMÜ: ARP Paketlerine İzin Ver
        Eğer ARP paketlerini (0x0806) engellerseniz, 15-60 dakika sonra router (yönlendirici) 
        sunucunun MAC adresini unutur. ARP isteklerine XDP seviyesinde DROP atıldığı için 
        sunucu cevap veremez ve ağdan tamamen düşer (Sanki herkesi banlamış gibi görünür).
    */
    if (eth->h_proto == __constant_htons(ETH_P_ARP))
        return XDP_PASS;

    struct PacketInfo packet = {};

    if (!ParsePacketByCTX(ctx, &packet)) {
        return XDP_DROP;
    }

    
    
//# 146:PASS dns 
__u64 *IsFound_dns = bpf_map_lookup_elem(&dns, &packet.IP);
if (IsFound_dns) {
		Uniq(&packet);
		Stat(&packet, 1);
		Log(&packet, 0x70646E73, 1);
 
	return XDP_PASS;
}
//# 148:IF packet.Protocol==17
if( packet.Protocol==17 ){
//# 149:DROP 
		Uniq(&packet);
		Stat(&packet, 0);
		Log(&packet, 0x55445020, 0);
		return XDP_DROP;
//# 150:END
}

//# 156:NEXT enableport 
__u32 Port_enableport = __builtin_bswap16(packet.InPort);
__u8 *IsFound_enableport = bpf_map_lookup_elem(&enableport, &Port_enableport);
if (IsFound_enableport && *IsFound_enableport==1) {}else{
		Uniq(&packet);
		Stat(&packet, 0);
		Log(&packet, 0x504F5254, 0);

	return XDP_DROP;
}

//# 159:GUARD FragmentAttack fragmented 120
if (packet.IsFragmented) {
	Log(&packet, 0x46524147, 0); // "FRAG"	
	return XDP_DROP;
}
//# 169:IF packet.OtPort == 80
if( packet.OtPort == 80 ){
//# 170:PASS
		Uniq(&packet);
		Stat(&packet, 1);
		return XDP_PASS;
//# 171:END
}

//# 174:DROP synblock 
__u64 *IsFound_synblock = bpf_map_lookup_elem(&synblock, &packet.IP);
if (IsFound_synblock) {
	__u64 now = bpf_ktime_get_ns();
	if (*IsFound_synblock > now){
		Uniq(&packet);
		Stat(&packet, 0);
		Log(&packet, 0x73796E63, 0);

		return XDP_DROP;
	}else{
		bpf_map_delete_elem(&synblock, &packet.IP);
	}
}

//# 175:GUARD SYNFloodAttack synblock 120
if (packet.Protocol == IPPROTO_TCP) {
		if (!CheckTCPFlags(&packet.TCP)) {
				//log_tcp_anomaly_drop(packet.Length);
				//send_log(0, &packet); // Test için anında logla
				return XDP_DROP; // Geçersiz TCP Bayrak Kombinasyonu (Nmap Stealth Scan vs.)
		}

		// SYN Flood Hız Sınırı (Rate Limit)
		if (packet.TCP.Flags & 0x02) { // 0x02 == SYN
				__u32 syn_key = 0;
				__u64 *global_syn = bpf_map_lookup_elem(&SYN_STATS, &syn_key);
				if (global_syn) {
						__sync_fetch_and_add(global_syn, 1);
						
						// Panic Mode: If Global SYN > 3000 per second
						// (bozkurt_daemon will reset this to 0 every second)
						if (*global_syn > 3000) {
								// Check for MSS Option (Basic OS Fingerprint)
								// If the header length is just 20 (no options), drop.
								if ((packet.TCP.DoL >> 4) < 6) { 
										return XDP_DROP; // Drop spoofed/dumb SYN
								}
								
								// Skip LRU map update to save CPU in panic mode
								// Pass to Kernel SYN Cookies
								goto skip_lru;
						}
				}

				struct syn_tracker_value *syn_val = bpf_map_lookup_elem(&syn_rate_limit, &packet.IP);
				__u64 now = bpf_ktime_get_ns();
				if (syn_val) {
						if (now - syn_val->first_seen > 1000000000ULL) { // 1 Saniye dolduysa sıfırla
								syn_val->count = 1;
								syn_val->first_seen = now;
						} else {
								syn_val->count++;
								if (syn_val->count > 50) { // Saniyede 30 SYN'den fazlaysa
										__u64 guard_syn_flood_attack_time_out= now + 120000000000;
										bpf_map_update_elem(&synblock, &packet.IP, &guard_syn_flood_attack_time_out, BPF_ANY);
										
										//log_syn_flood_drop(packet.Length);
										//send_log(0, &packet); // Test için anında logla
										return XDP_DROP;
								}
						}
				} else {
						struct syn_tracker_value new_val = {};
						new_val.count = 1;
						new_val.first_seen = now;
						bpf_map_update_elem(&syn_rate_limit, &packet.IP, &new_val, BPF_ANY);
				}
skip_lru:
				;
		}
}
	
//# 178:IF SYNC.mode
if( SYNC.mode ){

//# 179:PASS white 
__u64 *IsFound_white = bpf_map_lookup_elem(&white, &packet.IP);
if (IsFound_white) {
	__u64 now = bpf_ktime_get_ns();
	if (*IsFound_white > now){
		Uniq(&packet);
		Stat(&packet, 1);
		Log(&packet, 0x77686974, 1);
 
		return XDP_PASS;
	}else{
		bpf_map_delete_elem(&white, &packet.IP);
	}
}
//# 180:DROP
		Uniq(&packet);
		Stat(&packet, 0);
		return XDP_DROP;
//# 181:ELSE
}else{

//# 182:PASS white 
__u64 *IsFound_white = bpf_map_lookup_elem(&white, &packet.IP);
if (IsFound_white) {
	__u64 now = bpf_ktime_get_ns();
	if (*IsFound_white > now){
		Uniq(&packet);
		Stat(&packet, 1);
		Log(&packet, 0x77686974, 1);
 
		return XDP_PASS;
	}else{
		bpf_map_delete_elem(&white, &packet.IP);
	}
}

//# 183:DROP block 
__u64 *IsFound_block = bpf_map_lookup_elem(&block, &packet.IP);
if (IsFound_block) {
	__u64 now = bpf_ktime_get_ns();
	if (*IsFound_block > now){
		Uniq(&packet);
		Stat(&packet, 0);
		Log(&packet, 0x626C6F63, 0);

		return XDP_DROP;
	}else{
		bpf_map_delete_elem(&block, &packet.IP);
	}
}
//# 184:PASS 
		Uniq(&packet);
		Stat(&packet, 1);
		Log(&packet, 0x50415353, 1);
		return XDP_PASS;
//# 185:END
}
//# 188:PASS 
		Uniq(&packet);
		Stat(&packet, 1);
		Log(&packet, 0x70617373, 1);
		return XDP_PASS;


}

char LICENSE[] SEC("license") = "MIT";
