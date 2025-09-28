#!/usr/bin/env bash
set -euo pipefail

if [[ ${TPG_DEBUG:-0} -eq 1 ]]; then
  set -x
fi

if [[ "$(uname -s)" != "Linux" ]]; then
  echo "[TPG] 仅支持在 Linux 系统上运行该安装脚本" >&2
  exit 1
fi

if ! command -v curl >/dev/null 2>&1; then
  echo "[TPG] 未检测到 curl，请先安装 curl 再重试" >&2
  exit 1
fi

ARCH=$(uname -m)
case "${ARCH}" in
  x86_64|amd64)
    FILE_NAME="tpg-linux-amd64"
    ;;
  aarch64|arm64)
    FILE_NAME="tpg-linux-arm64"
    ;;
  armv7l|arm)
    FILE_NAME="tpg-linux-arm"
    ;;
  i386|i686)
    FILE_NAME="tpg-linux-386"
    ;;
  *)
    echo "[TPG] 暂不支持的 CPU 架构: ${ARCH}" >&2
    echo "[TPG] 支持的架构: x86_64(amd64), aarch64(arm64), armv7l(arm), i386(386)" >&2
    echo "[TPG] 请参考 README 手动下载对应的二进制文件" >&2
    exit 1
    ;;
esac

INSTALL_DIR=${INSTALL_DIR:-/usr/local/bin}
if [[ ! -w "${INSTALL_DIR}" ]]; then
  if command -v sudo >/dev/null 2>&1; then
    SUDO_PREFIX="sudo"
  else
    echo "[TPG] 目录 ${INSTALL_DIR} 需要写入权限，请使用 sudo 或设置 INSTALL_DIR 环境变量指向可写目录" >&2
    exit 1
  fi
else
  SUDO_PREFIX=""
fi

TMP_DIR=$(mktemp -d)
trap 'rm -rf "${TMP_DIR}"' EXIT

DOWNLOAD_URL="https://github.com/yinxulai/ait/releases/latest/download/${FILE_NAME}"
TARGET_PATH="${TMP_DIR}/tpg"

echo "[TPG] 检测到架构: ${ARCH}"
echo "[TPG] 正在从 ${DOWNLOAD_URL} 下载最新版本..."
curl -fsSL "${DOWNLOAD_URL}" -o "${TARGET_PATH}"
chmod +x "${TARGET_PATH}"

${SUDO_PREFIX} mkdir -p "${INSTALL_DIR}"
${SUDO_PREFIX} mv "${TARGET_PATH}" "${INSTALL_DIR}/tpg"

if command -v tpg >/dev/null 2>&1; then
  INSTALLED_PATH="$(command -v tpg)"
else
  INSTALLED_PATH="${INSTALL_DIR}/tpg"
fi

echo "[TPG] 安装成功 ✅"
echo "[TPG] 可执行文件位置: ${INSTALLED_PATH}"
echo "[TPG] 现在可以运行 'tpg -help' 查看完整参数"
