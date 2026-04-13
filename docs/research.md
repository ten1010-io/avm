# SR-IOV & P-Key 리서치 노트

> 2026-04-13 리서치 내용 정리
> AVM (Advanced Virtualization Manager) 개발 과정에서 학습한 내용

---

## 1. SR-IOV (Single Root I/O Virtualization)

### 1-1. 개념

SR-IOV는 **하나의 물리 장치를 여러 개의 가상 장치로 쪼개서 VM/Pod에 직접 연결해주는 기술**이다. 읽는 법은 "에스-알-아이-오-브이".

핵심 구조:
- **PF (Physical Function)**: 실제 물리 장치 (예: 10Gbps NIC 포트 1개)
- **VF (Virtual Function)**: PF에서 쪼개진 가상 장치. VM/Pod에 직접 할당 가능

```
일반 방식: VM → 하이퍼바이저(관리자) → NIC      (느림, CPU 사용 높음)
SR-IOV:    VM → VF → NIC 직통                   (빠름, CPU 거의 안 씀)
```

### 1-2. NVIDIA MIG와의 차이

| 구분 | SR-IOV | NVIDIA MIG |
|------|--------|------------|
| 쪼개는 대상 | I/O 인터페이스 (데이터 통로) | GPU 연산 자원 (코어, 메모리) |
| 분리 방식 | 논리적 분할 (차선 나누기) | 물리적 격리 (벽이 있는 방 나누기) |
| 목적 | 데이터 전송 속도 향상, 지연 감소 | 자원 낭비 방지, 안정성 확보 |
| Noisy Neighbor | **발생 가능** (대역폭 공유) | **발생 불가** (물리적 격리) |
| 대상 장치 | NIC, GPU 등 PCIe 장치 | A100, H100 등 고성능 GPU |

### 1-3. Noisy Neighbor 문제

SR-IOV는 논리적 분할이기 때문에 **물리 대역폭을 공유**한다. 한 VM이 대역폭을 독식하면 다른 VM들이 피해를 받는다.

```
10Gbps NIC에 VF 64개 생성 → 동시 사용 시 VM당 ~156Mbps
VF 1개가 9.9Gbps 독식 → 나머지 63개가 100Mbps를 나눠먹음
```

**해결 방법**: QoS (Quality of Service) 설정

```bash
# 대역폭 제한 (상한선)
ip link set ens3f0 vf 0 max_tx_rate 5000   # 최대 5Gbps

# 최소 보장 (하한선)
ip link set ens3f0 vf 0 min_tx_rate 1000   # 최소 1Gbps 보장

# VLAN + 우선순위
ip link set ens3f0 vf 0 vlan 100 qos 7     # 최고 우선순위 (0~7)
```

| 설정 | 역할 | 비유 |
|------|------|------|
| `max_tx_rate` | 최대 속도 제한 | 속도 제한 표지판 |
| `min_tx_rate` | 최소 대역폭 보장 | 전용 차선 확보 |
| `vlan qos` | 트래픽 우선순위 | 긴급차량 우선 통행 |

기본값은 아무 제한 없이 **선착순 나눠먹기(Best Effort)**이다. 우선순위는 관리자가 직접 설정해야 한다.

### 1-4. VF와 VM/Pod의 관계

- **VF 1개 = VM/Pod 1개** (1:1 관계, 한 VF에 여러 VM 물릴 수 없음)
- VF 생성 ≠ VM에 자동 할당. VF는 "빈 하이패스 게이트"를 만들어 두는 것
- VM/Pod에 연결하는 건 별도 작업 (libvirt, K8s SR-IOV Device Plugin 등)
- 이미 할당된 VF를 다른 VM에 물리려고 하면 에러 발생

### 1-5. SR-IOV 도입 체크리스트 (4단계)

| 단계 | 확인 내용 | 확인 방법 |
|------|----------|----------|
| 1. 장치 지원 | NIC/GPU가 SR-IOV 지원하는지 | `lspci -v \| grep "Single Root I/O Virtualization"` |
| 2. BIOS 설정 | IOMMU(VT-d/AMD-Vi) + SR-IOV 활성화 | BIOS 진입 → Advanced → PCIe 설정 |
| 3. 커널 파라미터 | IOMMU 활성화 확인 | `cat /proc/cmdline`에서 `intel_iommu=on` 확인 |
| 4. VF 개수 산정 | Max VF 대비 적정 수 설정 | `cat /sys/bus/pci/devices/{bdf}/sriov_totalvfs` |

### 1-6. SR-IOV가 의미 있는 워크로드

| 상황 | SR-IOV 필요? | 이유 |
|------|-------------|------|
| 일반 웹 서비스 | ✗ | 기존 CNI로 충분 |
| REST API 서버 | ✗ | 네트워크가 병목이 아님 |
| ML 학습 (GPU간 통신) | **✓** | RDMA/GPUDirect에 저지연 필수 |
| DPDK 패킷 처리 | **✓** | 커널 bypass가 핵심 |
| 고빈도 금융 트레이딩 | **✓** | 마이크로초 단위 지연 중요 |
| 대용량 스토리지 I/O | **✓** | NVMe-oF 등 고대역폭 필요 |
| 일반 컨테이너 워크로드 | ✗ | 오버엔지니어링 |

성능 비교:
```
일반 CNI (Calico/Flannel):  지연 ~100μs,  처리량 ~5Gbps,  CPU 사용 높음
SR-IOV:                     지연 ~10μs,   처리량 ~25Gbps, CPU 사용 거의 없음
```

---

## 2. IOMMU (Input-Output Memory Management Unit)

### 2-1. 개념

IOMMU는 물리 장치의 메모리 주소를 VM에 직접 매핑해주는 하드웨어 기능이다. SR-IOV가 동작하려면 IOMMU가 반드시 활성화되어야 한다.

| CPU 벤더 | IOMMU 이름 | 커널 파라미터 |
|----------|-----------|-------------|
| Intel | VT-d (VT-x와 다름. VT-x는 CPU 가상화, VT-d가 I/O 가상화) | `intel_iommu=on` |
| AMD | AMD-Vi | `amd_iommu=on` |

### 2-2. IOMMU 3가지 상태

실제 회사 서버(RHEL 8)에서 테스트해보니 IOMMU는 단순 on/off가 아니라 3단계 상태가 있었다.

| 상태 | 의미 | 감지 방법 |
|------|------|----------|
| **Not Supported** | BIOS에서 VT-d 자체가 꺼져있음 | dmesg에 IOMMU 관련 메시지 없음 |
| **Passthrough** | HW 지원하지만 커널이 passthrough 모드로 동작 | dmesg: "Default domain type: Passthrough" |
| **Enabled** | 완전 활성화, SR-IOV 사용 가능 | `/sys/kernel/iommu_groups/`에 그룹 존재 |

회사 서버 확인 결과:
```bash
$ dmesg | grep -i iommu
[1.112026] iommu: Default domain type: Passthrough

$ cat /proc/cmdline
# intel_iommu=on 없었음

$ ls /sys/kernel/iommu_groups/
# 비어있음
```
→ BIOS에서 VT-d는 켜져있지만, 커널 파라미터가 없어서 Passthrough 모드였음

### 2-3. Passthrough → Enabled 전환

1. `/etc/default/grub`의 `GRUB_CMDLINE_LINUX`에 `intel_iommu=on iommu=pt` 추가
2. `grub2-mkconfig -o /boot/grub2/grub.cfg` 실행 (CentOS/RHEL)
   또는 `update-grub` (Ubuntu)
3. **서버 재부팅 필요** (커널 파라미터는 부팅 시점에 읽히므로 런타임 반영 불가)

---

## 3. Kubernetes 환경에서의 SR-IOV

### 3-1. 일반 CNI vs SR-IOV 경로 비교

```
일반 CNI:  Pod → veth → bridge/overlay → 호스트 커널 네트워크 스택 → 물리 NIC
           패킷이 거치는 단계: 5~6단계, CPU가 모든 패킷 처리

SR-IOV:    Pod → VF → 물리 NIC (직통)
           패킷이 거치는 단계: 1~2단계, CPU 거의 안 씀
```

### 3-2. K8s SR-IOV 구성 3단계

VF 생성부터 Pod 사용까지 3개의 CRD를 거친다:

```
Step 1: SriovNetworkNodePolicy  → VF 생성 + 풀(등급) 분류
Step 2: SriovNetwork            → QoS/VLAN/rate limiting 설정
Step 3: Pod spec                → 원하는 네트워크에 연결 요청
```

#### Step 1: VF 풀 분류 (SriovNetworkNodePolicy)

VF를 등급별로 나눠 별도 리소스 이름으로 등록한다.

```yaml
apiVersion: sriovnetwork.openshift.io/v1
kind: SriovNetworkNodePolicy
metadata:
  name: high-perf-policy
  namespace: sriov-network-operator
spec:
  resourceName: highperf_vf       # Pod가 요청할 리소스 이름
  numVfs: 4
  nicSelector:
    pfNames: ["ens3f0#0-1"]       # VF 0~1번만 이 풀에 포함
  deviceType: netdevice
  nodeSelector:
    sriov: "true"
```

#### Step 2: QoS 설정 (SriovNetwork)

**중요**: QoS/rate limiting은 SriovNetworkNodePolicy가 아니라 **SriovNetwork**에서 설정한다.

```yaml
apiVersion: sriovnetwork.openshift.io/v1
kind: SriovNetwork
metadata:
  name: high-perf-net
  namespace: sriov-network-operator
spec:
  resourceName: highperf_vf       # 위 Policy와 연결
  networkNamespace: default
  vlan: 100
  minTxRate: 1000                  # 최소 1Gbps 보장
  maxTxRate: 5000                  # 최대 5Gbps 제한
  spoofChk: "on"
  trust: "off"
```

| SriovNetwork 필드 | 역할 | 예시 |
|-------------------|------|------|
| `maxTxRate` | 최대 대역폭 (Mbps) | `5000` = 5Gbps |
| `minTxRate` | 최소 보장 대역폭 (Mbps) | `1000` = 1Gbps |
| `vlan` | VLAN ID | `100` |
| `spoofChk` | MAC 스푸핑 방지 | `"on"` / `"off"` |
| `trust` | VF 신뢰 모드 | `"on"` / `"off"` |

#### Step 3: Pod에서 사용

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: ml-worker
  annotations:
    k8s.v1.cni.cncf.io/networks: high-perf-net   # SriovNetwork 이름
spec:
  containers:
  - name: worker
    resources:
      requests:
        openshift.io/highperf_vf: "1"             # 리소스 요청
      limits:
        openshift.io/highperf_vf: "1"
```

### 3-3. VF → Pod 매핑 문제와 해결

기본 SR-IOV Device Plugin은 VF 풀에서 **랜덤으로** 할당한다. QoS가 다른 VF가 섞여 있으면 의도한 성능이 안 나올 수 있다.

**해결**: VF를 등급별 풀로 나누고, Pod가 원하는 풀의 resourceName을 지정 요청한다.

```
VF 풀 구성 예:
├── highperf_vf   (VF 0~1, max 5Gbps)  → ML 학습 Pod가 요청
├── standard_vf   (VF 2~3, max 1Gbps)  → 일반 Pod가 요청
```

QoS를 VF 개별로 `ip link`로 수동 설정하는 것이 아니라, SriovNetworkNodePolicy로 풀을 나누고 SriovNetwork에서 풀 단위로 QoS를 건다. 이렇게 해야 Pod 재생성 시에도 QoS가 유지된다.

---

## 4. P-Key (Partition Key)

### 4-1. 개념

P-Key는 InfiniBand 네트워크의 **파티션 키**로, 이더넷의 VLAN에 해당한다. 같은 P-Key를 가진 장치끼리만 통신할 수 있어 트래픽을 격리한다.

### 4-2. SR-IOV + P-Key를 함께 사용하는 이유

| 조합 | 결과 |
|------|------|
| SR-IOV만 | 빠르지만 모든 Pod이 같은 네트워크에서 섞임 |
| P-Key만 | 격리되지만 커널 스택 거쳐서 느림 |
| **둘 다** | **빠르고 + 격리됨** (ML 멀티테넌트 환경의 정석) |

```
물리 서버 (ConnectX-6 NIC)
├── VF 0 (P-Key 0x8001) → 팀A ML Pod  ← 팀A끼리만 RDMA 통신
├── VF 1 (P-Key 0x8001) → 팀A ML Pod
├── VF 2 (P-Key 0x8002) → 팀B ML Pod  ← 팀B끼리만 RDMA 통신
└── VF 3 (P-Key 0x8002) → 팀B ML Pod
```

### 4-3. P-Key 생성 흐름

P-Key는 **Subnet Manager(SM)에서 먼저 생성**해야 한다. K8s CRD는 이미 존재하는 P-Key를 VF에 연결해주는 것뿐이다. SM에 없는 P-Key를 CRD에 넣으면 에러가 난다.

```
Step 1: Subnet Manager (OpenSM/UFM) → P-Key 파티션 생성
Step 2: ib-kubernetes 플러그인      → VF에 P-Key 매핑
Step 3: SriovNetwork CRD            → Pod 연결 시 P-Key 지정
```

전체 그림:
```
                    ┌──────────────────┐
                    │  Subnet Manager  │
                    │  (OpenSM / UFM)  │
                    │                  │
                    │  P-Key 0x8001    │  ← Step 1: 여기서 파티션 생성
                    │  P-Key 0x8002    │
                    └────────┬─────────┘
                             │
                    ┌────────▼─────────┐
                    │  ib-kubernetes   │
                    │  (DaemonSet)     │  ← Step 2: VF에 P-Key 매핑
                    └────────┬─────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
         ┌────▼────┐   ┌────▼────┐   ┌────▼────┐
         │  VF 0   │   │  VF 1   │   │  VF 2   │
         │ 0x8001  │   │ 0x8001  │   │ 0x8002  │
         │ 팀A Pod │   │ 팀A Pod │   │ 팀B Pod │
         └─────────┘   └─────────┘   └─────────┘
```

#### OpenSM 설정 (partitions.conf)

```
# /etc/opensm/partitions.conf

# team-a-ml
0x8001 : ALL=full ;

# team-b-ml
0x8002 : ALL=full ;
```

#### K8s에서 P-Key 연결 (SriovNetwork)

```yaml
apiVersion: sriovnetwork.openshift.io/v1
kind: SriovNetwork
metadata:
  name: team-a-rdma
spec:
  resourceName: mlnx_rdma_vf
  networkNamespace: team-a
  metaPlugins: |
    {
      "type": "ib-kubernetes",
      "pkey": "0x8001"
    }
```

### 4-4. OpenSM vs UFM

| | OpenSM | UFM (NVIDIA) |
|---|--------|-------------|
| 가격 | 무료 (오픈소스) | 유료 라이센스 |
| 설치 | `yum install opensm` / `apt install opensm` | 별도 서버에 설치 |
| 설정 | conf 파일 수동 편집 | 웹 UI |
| 규모 | 소규모~중규모 (수십 노드) | 대규모 (수백~수천 노드) |

GPU 노드 10대 이하면 OpenSM으로 충분. OpenSM은 InfiniBand 표준 기반이라 NVIDIA(Mellanox) 장비에서도 동작한다.

### 4-5. P-Key의 하드웨어 요구사항

**InfiniBand NIC(Mellanox ConnectX 등)이 물리적으로 장착되어 있어야 한다.**

```
OpenSM 설치 → 소프트웨어라 가능
OpenSM 실행 → IB 포트 못 찾으면 에러
P-Key 생성 → IB 포트 없으면 의미 없음
macOS      → RDMA 스택 자체가 없어 OpenSM 설치조차 불가
```

IB 장치 확인:
```bash
ibstat                          # IB 장치 상태
ls /sys/class/infiniband/       # sysfs에서 IB 장치 목록
```

---

## 5. 역할 분담 정리

### AVM TUI가 담당하는 영역 (인프라 준비 + 모니터링)

| 기능 | 설명 |
|------|------|
| IOMMU 상태 확인 | Enabled / Passthrough / Not Supported 3단계 감지 |
| IOMMU 활성화 | GRUB 파라미터 수정 (재부팅 필요) |
| VF 생성/삭제 | `sriov_numvfs` 읽기/쓰기 |
| VF 상태 모니터링 | 드라이버, MAC, Pod 할당 상태, QoS 정보 |
| P-Key 생성/삭제 | OpenSM partitions.conf 관리 |

### K8s CRD가 담당하는 영역 (AVM 범위 밖)

| CRD | 역할 |
|-----|------|
| SriovNetworkNodePolicy | VF 풀 분류, resourceName 지정 |
| SriovNetwork | QoS/rate limiting, VLAN, P-Key 매핑 |
| Pod spec | 원하는 리소스/네트워크 요청 |

---

## 6. 환경별 제약 사항

| 기능 | macOS | Linux (이더넷만) | Linux (IB NIC 있음) |
|------|-------|-----------------|-------------------|
| IOMMU 감지 | ✗ | ✓ | ✓ |
| VF 생성 | ✗ | ✓ (SR-IOV NIC 필요) | ✓ |
| P-Key 관리 | ✗ | ✗ | ✓ |
| OpenSM 설치 | ✗ | 설치 가능하나 IB 없으면 무의미 | ✓ |
| AVM --demo | ✓ (UI 확인용) | ✓ | ✓ |

---

## 7. 주요 sysfs / 명령어 레퍼런스

| 용도 | 경로 / 명령어 |
|------|-------------|
| IOMMU 그룹 | `/sys/kernel/iommu_groups/` |
| 커널 파라미터 | `/proc/cmdline` |
| IOMMU dmesg | `dmesg \| grep -i iommu` |
| SR-IOV 디바이스 검색 | `/sys/bus/pci/devices/*/sriov_totalvfs` |
| 현재 VF 수 | `/sys/bus/pci/devices/{bdf}/sriov_numvfs` |
| VF 설정 | `echo N > /sys/bus/pci/devices/{bdf}/sriov_numvfs` |
| 벤더/디바이스 ID | `/sys/bus/pci/devices/{bdf}/vendor`, `device` |
| 드라이버 | `/sys/bus/pci/devices/{bdf}/driver` (symlink) |
| VF 드라이버 확인 | `/sys/bus/pci/devices/{vf_bdf}/driver` |
| IB 장치 확인 | `ibstat`, `ls /sys/class/infiniband/` |
| Active P-Key 확인 | `/sys/class/infiniband/*/ports/*/pkeys/*` |
| lspci 상세 | `lspci -vmm -s {bdf}` |
| GRUB 설정 | `/etc/default/grub` |
| OpenSM 설정 | `/etc/opensm/partitions.conf` |

---

## 8. AVM 프로젝트 현황

- **GitHub**: https://github.com/ten1010-io/avm
- **풀네임**: Advanced Virtualization Manager
- **최신 릴리즈**: v0.2.0
- **기술 스택**: Go + bubbletea + lipgloss + bubbles
- **빌드 배포**: GitHub Actions 크로스 컴파일 (CGO_ENABLED=0 정적 빌드)
- **지원 바이너리**: linux-amd64, linux-arm64, darwin-amd64, darwin-arm64

서버에서 실행:
```bash
curl -LO https://github.com/ten1010-io/avm/releases/download/v0.2.0/avm-linux-amd64
chmod +x avm-linux-amd64
sudo ./avm-linux-amd64        # 실제 장비
./avm-linux-amd64 --demo      # 데모 모드
```
