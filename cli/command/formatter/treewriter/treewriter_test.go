package treewriter

import (
	"testing"
)

func TestTree(t *testing.T) {
	var rows [][]string
	rows = append(rows, [][]string{
		{"", "alpine:latest", "beefdbd8a1da", "13.6MB", "4.09MB"},
		{"├─", "linux/amd64", "33735bd63cf8", "0B", "0B"},
		{"├─", "linux/arm/v6", "50f635c8b04d", "0B", "0B"},
		{"├─", "linux/arm/v7", "f2f82d424957", "0B", "0B"},
		{"├─", "linux/arm64/v8", "9cee2b382fe2", "13.6MB", "4.09MB"},
		{"├─", "linux/386", "b3e87f642f5c", "0B", "0B"},
		{"├─", "linux/ppc64le", "c7a6800e3dc5", "0B", "0B"},
		{"├─", "linux/riscv64", "80cde017a105", "0B", "0B"},
		{"└─", "linux/s390x", "2b5b26e09ca2", "0B", "0B"},

		{"", "namespace/image", "beefdbd8a1da", "13.6MB", "4.09MB"},
		{"├─", "namespace/image:1", "beefdbd8a1da", "-", "-"},
		{"├─", "namespace/image:1.0", "beefdbd8a1da", "-", "-"},
		{"├─", "namespace/image:1.0.0", "beefdbd8a1da", "-", "-"},
		{"└─", "namespace/image:latest", "beefdbd8a1da", "-", "-"},
		{"   ├─", "linux/amd64", "33735bd63cf8", "0B", "0B"},
		{"   ├─", "linux/arm/v6", "50f635c8b04d", "0B", "0B"},
		{"   ├─", "linux/arm/v7", "f2f82d424957", "0B", "0B"},
		{"   ├─", "linux/arm64/v8", "9cee2b382fe2", "13.6MB", "4.09MB"},
		{"   ├─", "linux/386", "b3e87f642f5c", "0B", "0B"},
		{"   ├─", "linux/ppc64le", "c7a6800e3dc5", "0B", "0B"},
		{"   ├─", "linux/riscv64", "80cde017a105", "0B", "0B"},
		{"   └─", "linux/s390x", "2b5b26e09ca2", "0B", "0B"},

		{"", "internal.example.com/namespace/image", "beefdbd8a1da", "13.6MB", "4.09MB"},
		{"├─", "internal.example.com/namespace/image:1", "beefdbd8a1da", "-", "-"},
		{"├─", "internal.example.com/namespace/image:1.0", "beefdbd8a1da", "-", "-"},
		{"├─", "internal.example.com/namespace/image:1.0.0", "beefdbd8a1da", "-", "-"},
		{"└─", "internal.example.com/namespace/image:latest", "beefdbd8a1da", "-", "-"},
		{"   ├─", "linux/amd64", "33735bd63cf8", "0B", "0B"},
		{"   ├─", "linux/arm/v6", "50f635c8b04d", "0B", "0B"},
		{"   ├─", "linux/arm/v7", "f2f82d424957", "0B", "0B"},
		{"   ├─", "linux/arm64/v8", "9cee2b382fe2", "13.6MB", "4.09MB"},
		{"   ├─", "linux/386", "b3e87f642f5c", "0B", "0B"},
		{"   ├─", "linux/ppc64le", "c7a6800e3dc5", "0B", "0B"},
		{"   ├─", "linux/riscv64", "80cde017a105", "0B", "0B"},
		{"   └─", "linux/s390x", "2b5b26e09ca2", "0B", "0B"},
	}...)

	header := []string{"IMAGE/TAGS", "ID", "DISK USAGE", "CONTENT SIZE", "USED"}
	PrintTree(header, rows)
}

func TestTreeNoTrunc(t *testing.T) {
	var rows [][]string
	rows = append(rows, [][]string{
		{"", "alpine:latest", "sha256:beefdbd8a1da6d2915566fde36db9db0b524eb737fc57cd1367effd16dc0d06d", "13.6MB", "4.09MB"},
		{"├─", "linux/amd64", "sha256:33735bd63cf84d7e388d9f6d297d348c523c044410f553bd878c6d7829612735", "0B", "0B"},
		{"├─", "linux/arm/v6", "sha256:50f635c8b04d86dde8a02bcd8d667ba287eb8b318c1c0cf547e5a48ddadea1be", "0B", "0B"},
		{"├─", "linux/arm/v7", "sha256:f2f82d42495723c4dc508fd6b0978a5d7fe4efcca4282e7aae5e00bcf4057086", "0B", "0B"},
		{"├─", "linux/arm64/v8", "sha256:9cee2b382fe2412cd77d5d437d15a93da8de373813621f2e4d406e3df0cf0e7c", "13.6MB", "4.09MB"},
		{"├─", "linux/386", "sha256:b3e87f642f5c48cdc7556c3e03a0d63916bd0055ba6edba7773df3cb1a76f224", "0B", "0B"},
		{"├─", "linux/ppc64le", "sha256:c7a6800e3dc569a2d6e90627a2988f2a7339e6f111cdf6a0054ad1ff833e99b0", "0B", "0B"},
		{"├─", "linux/riscv64", "sha256:80cde017a10529a18a7274f70c687bb07c4969980ddfb35a1b921fda3a020e5b", "0B", "0B"},
		{"└─", "linux/s390x", "sha256:2b5b26e09ca2856f50ac88312348d26c1ac4b8af1df9f580e5cf465fd76e3d4d", "0B", "0B"},

		{"", "namespace/image", "sha256:beefdbd8a1da6d2915566fde36db9db0b524eb737fc57cd1367effd16dc0d06d", "13.6MB", "4.09MB"},
		{"├─", "namespace/image:1", "sha256:beefdbd8a1da6d2915566fde36db9db0b524eb737fc57cd1367effd16dc0d06d", "-", "-"},
		{"├─", "namespace/image:1.0", "sha256:beefdbd8a1da6d2915566fde36db9db0b524eb737fc57cd1367effd16dc0d06d", "-", "-"},
		{"├─", "namespace/image:1.0.0", "sha256:beefdbd8a1da6d2915566fde36db9db0b524eb737fc57cd1367effd16dc0d06d", "-", "-"},
		{"└─", "namespace/image:latest", "sha256:beefdbd8a1da6d2915566fde36db9db0b524eb737fc57cd1367effd16dc0d06d", "-", "-"},
		{"   ├─", "linux/amd64", "sha256:33735bd63cf84d7e388d9f6d297d348c523c044410f553bd878c6d7829612735", "0B", "0B"},
		{"   ├─", "linux/arm/v6", "sha256:50f635c8b04d86dde8a02bcd8d667ba287eb8b318c1c0cf547e5a48ddadea1be", "0B", "0B"},
		{"   ├─", "linux/arm/v7", "sha256:f2f82d42495723c4dc508fd6b0978a5d7fe4efcca4282e7aae5e00bcf4057086", "0B", "0B"},
		{"   ├─", "linux/arm64/v8", "sha256:9cee2b382fe2412cd77d5d437d15a93da8de373813621f2e4d406e3df0cf0e7c", "13.6MB", "4.09MB"},
		{"   ├─", "linux/386", "sha256:b3e87f642f5c48cdc7556c3e03a0d63916bd0055ba6edba7773df3cb1a76f224", "0B", "0B"},
		{"   ├─", "linux/ppc64le", "sha256:c7a6800e3dc569a2d6e90627a2988f2a7339e6f111cdf6a0054ad1ff833e99b0", "0B", "0B"},
		{"   ├─", "linux/riscv64", "sha256:80cde017a10529a18a7274f70c687bb07c4969980ddfb35a1b921fda3a020e5b", "0B", "0B"},
		{"   └─", "linux/s390x", "sha256:2b5b26e09ca2856f50ac88312348d26c1ac4b8af1df9f580e5cf465fd76e3d4d", "0B", "0B"},

		{"", "internal.example.com/namespace/image", "sha256:beefdbd8a1da6d2915566fde36db9db0b524eb737fc57cd1367effd16dc0d06d", "13.6MB", "4.09MB"},
		{"├─", "internal.example.com/namespace/image:1", "sha256:beefdbd8a1da6d2915566fde36db9db0b524eb737fc57cd1367effd16dc0d06d", "-", "-"},
		{"├─", "internal.example.com/namespace/image:1.0", "sha256:beefdbd8a1da6d2915566fde36db9db0b524eb737fc57cd1367effd16dc0d06d", "-", "-"},
		{"├─", "internal.example.com/namespace/image:1.0.0", "sha256:beefdbd8a1da6d2915566fde36db9db0b524eb737fc57cd1367effd16dc0d06d", "-", "-"},
		{"└─", "internal.example.com/namespace/image:latest", "sha256:beefdbd8a1da6d2915566fde36db9db0b524eb737fc57cd1367effd16dc0d06d", "-", "-"},
		{"   ├─", "linux/amd64", "sha256:33735bd63cf84d7e388d9f6d297d348c523c044410f553bd878c6d7829612735", "0B", "0B"},
		{"   ├─", "linux/arm/v6", "sha256:50f635c8b04d86dde8a02bcd8d667ba287eb8b318c1c0cf547e5a48ddadea1be", "0B", "0B"},
		{"   ├─", "linux/arm/v7", "sha256:f2f82d42495723c4dc508fd6b0978a5d7fe4efcca4282e7aae5e00bcf4057086", "0B", "0B"},
		{"   ├─", "linux/arm64/v8", "sha256:9cee2b382fe2412cd77d5d437d15a93da8de373813621f2e4d406e3df0cf0e7c", "13.6MB", "4.09MB"},
		{"   ├─", "linux/386", "sha256:b3e87f642f5c48cdc7556c3e03a0d63916bd0055ba6edba7773df3cb1a76f224", "0B", "0B"},
		{"   ├─", "linux/ppc64le", "sha256:c7a6800e3dc569a2d6e90627a2988f2a7339e6f111cdf6a0054ad1ff833e99b0", "0B", "0B"},
		{"   ├─", "linux/riscv64", "sha256:80cde017a10529a18a7274f70c687bb07c4969980ddfb35a1b921fda3a020e5b", "0B", "0B"},
		{"   └─", "linux/s390x", "sha256:2b5b26e09ca2856f50ac88312348d26c1ac4b8af1df9f580e5cf465fd76e3d4d", "0B", "0B"},
	}...)

	header := []string{"IMAGE/TAGS", "ID", "DISK USAGE", "CONTENT SIZE", "USED"}
	PrintTree(header, rows)
}
