FFmpeg 8.1.2 uses code under LGPL-2.1-or-later and optional files/components under GPL-2.0-or-later. The Alpine v3.24 ffmpeg package used by Snitch is declared GPL-2.0-or-later AND LGPL-2.1-or-later. See LICENSE.md, COPYING.LGPLv2.1 and COPYING.GPLv2 for the upstream license texts.

Upstream source tag: https://github.com/FFmpeg/FFmpeg/tree/n8.1.2
Upstream source archive: https://ffmpeg.org/releases/ffmpeg-8.1.2.tar.xz
Alpine v3.24 aports package recipe/build context: https://gitlab.alpinelinux.org/alpine/aports/-/tree/d2c3ca384892f415fc1b92abd87f78c1d5d0cbba/community/ffmpeg
Alpine arm64 package metadata: https://pkgs.alpinelinux.org/package/v3.24/community/aarch64/ffmpeg
Alpine x86_64 package metadata: https://pkgs.alpinelinux.org/package/v3.24/community/x86_64/ffmpeg

The image installs ffmpeg=8.1.2-r0 from Alpine v3.24; ffprobe is used as an independent executable and is not linked into the Snitch Go binary. The package's exact source and build configuration are available through the Alpine recipe and source archives linked above. Recheck source correspondence and notices before publishing an image.
