# AVM (Advanced Virtualization Manager) - TUI Application Plan

## Context
SR-IOV VF 관리를 위한 Go + bubbletea 기반 TUI 애플리케이션. 현재 장비의 IOMMU 지원 여부, SR-IOV 디바이스 목록, Max VF 확인, VF 개수 설정 등을 터미널에서 직관적으로 수행할 수 있는 도구.

## Tech Stack
- **Go** + `bubbletea` (TUI framework) + `lipgloss` (styling) + `bubbles` (UI components)
- **sysfs** 직접 읽기 (PCI 디바이스 정보, SR-IOV 상태)
- `lspci` fallback (vendor/device 이름 조회)

## Project Structure
```
avm/
├── main.go                      # Entry point
├── go.mod
├── internal/
│   ├── tui/
│   │   ├── app.go               # Root model (view routing)
│   │   ├── styles.go            # lipgloss 스타일 정의
│   │   ├── dashboard.go         # 메인 대시보드 (IOMMU 상태 + 디바이스 목록)
│   │   ├── device_detail.go     # 디바이스 상세 + VF 설정 폼
│   │   └── messages.go          # 커스텀 메시지 타입
│   └── sriov/
│       ├── iommu.go             # IOMMU 지원 감지
│       ├── device.go            # SR-IOV 디바이스 스캔
│       ├── vf.go                # VF 개수 조회/설정
│       └── demo.go              # 데모 데이터 (macOS 테스트용)
```

## 화면 구성

### 1. Dashboard (메인 화면)
```
╭─ AVM - Advanced Virtualization Manager ─────────────────────╮
│                                                   │
│  IOMMU Status: ✓ Enabled (intel_iommu=on)        │
│                                                   │
│  SR-IOV Capable Devices:                          │
│  ┌────────┬──────────┬───────────┬────┬─────┐    │
│  │ BDF    │ Device   │ Driver    │ VFs│ Max │    │
│  ├────────┼──────────┼───────────┼────┼─────┤    │
│  │ 03:00.0│ X710 NIC │ i40e      │ 0  │ 64  │    │
│  │ 82:00.0│ A100 GPU │ nvidia    │ 4  │ 16  │    │
│  └────────┴──────────┴───────────┴────┴─────┘    │
│                                                   │
│  [Enter] Detail  [q] Quit                        │
╰──────────────────────────────────────────────────╯
```

IOMMU 미지원 시:
```
│  IOMMU Status: ✗ Not Enabled                     │
│  ⚠ SR-IOV requires IOMMU. Enable in BIOS:       │
│    Intel: VT-d  /  AMD: AMD-Vi                   │
│    Then add intel_iommu=on to kernel params       │
```

### 2. Device Detail (디바이스 상세 + VF 설정)
```
╭─ Device: 0000:03:00.0 - Intel X710 ─────────────╮
│                                                   │
│  Vendor:  Intel Corporation                       │
│  Driver:  i40e                                    │
│  Class:   Ethernet controller                     │
│                                                   │
│  VF Configuration:                                │
│  Current VFs: 0 / Max: 64                         │
│                                                   │
│  Set VF Count: [    4    ]                        │
│                                                   │
│  [Enter] Apply  [Esc] Back                       │
╰──────────────────────────────────────────────────╯
```

## Implementation Phases

### Phase 1: 프로젝트 초기화 + sriov 백엔드
1. `go mod init github.com/joonseolee/avm`
2. `internal/sriov/iommu.go` - IOMMU 감지
   - `/sys/kernel/iommu_groups/` 디렉토리 존재 및 내용 확인
   - `/proc/cmdline`에서 `intel_iommu=on` / `amd_iommu=on` 파싱
3. `internal/sriov/device.go` - SR-IOV 디바이스 스캔
   - `/sys/bus/pci/devices/*/sriov_totalvfs` 존재하는 디바이스 목록
   - vendor/device ID → 이름 매핑 (`lspci -vmm -s` 또는 `/usr/share/hwdata/pci.ids`)
4. `internal/sriov/vf.go` - VF 관리
   - `sriov_numvfs` 읽기/쓰기 (쓰기는 root 필요)

### Phase 2: TUI 구현
5. `internal/tui/styles.go` - 공통 스타일
6. `internal/tui/messages.go` - 커스텀 메시지 타입
7. `internal/tui/dashboard.go` - IOMMU 상태 + 디바이스 테이블
8. `internal/tui/device_detail.go` - 상세 뷰 + VF 설정 폼
9. `internal/tui/app.go` - 뷰 라우팅 (dashboard ↔ detail)
10. `main.go` - 진입점

## Key Dependencies
```
github.com/charmbracelet/bubbletea
github.com/charmbracelet/lipgloss
github.com/charmbracelet/bubbles
```

## sysfs 접근 경로
| 용도 | 경로 |
|------|------|
| IOMMU 그룹 | `/sys/kernel/iommu_groups/` |
| 커널 파라미터 | `/proc/cmdline` |
| SR-IOV 디바이스 | `/sys/bus/pci/devices/*/sriov_totalvfs` |
| 현재 VF 수 | `/sys/bus/pci/devices/{bdf}/sriov_numvfs` |
| VF 설정 | `echo N > /sys/bus/pci/devices/{bdf}/sriov_numvfs` |
| 벤더 ID | `/sys/bus/pci/devices/{bdf}/vendor` |
| 디바이스 ID | `/sys/bus/pci/devices/{bdf}/device` |
| 드라이버 | `/sys/bus/pci/devices/{bdf}/driver` (symlink) |

## Verification
1. `go build -o avm .` 빌드 확인
2. macOS에서는 sysfs 없으므로 mock/demo 모드 필요 → `--demo` 플래그로 샘플 데이터 표시
3. Linux에서 실제 테스트: `sudo ./avm` (VF 설정은 root 필요)
