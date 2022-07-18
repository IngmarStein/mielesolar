#!/bin/sh
SPK_PACKAGE_FILE="mielesolar-${SPK_ARCH:-x86_64}-${SPK_PACKAGE_SUFFIX:-latest}.spk"
cat <<EOF
{
  "packages": [
    {
      "package": "mielesolar",
      "version": "${SPK_PACKAGE_VERSION:-1.0.0}",
      "dname": "Mielesolar",
      "desc": "Mielesolar on Synology DSM.",
      "price": 0,
      "download_count": 56691,
      "recent_download_count": 1138,
      "link": "${SPK_PACKAGE_URL}",
      "size": $(stat -f "%z" "${SPK_PACKAGE_FILE}"),
      "md5": "$(md5sum "${SPK_PACKAGE_FILE}")",
      "snapshot": [],
      "qinst": true,
      "qstart": true,
      "qupgrade": true,
      "depsers": null,
      "deppkgs": null,
      "conflictpkgs": null,
      "start": true,
      "maintainer": "IngmarStein",
      "maintainer_url": "https://github.com/IngmarStein/mielesolar",
      "distributor": "",
      "distributor_url": "",
      "support_url": "",
      "changelog": "",
      "thirdparty": true,
      "category": 0,
      "subcategory": 0,
      "type": 0,
      "silent_install": false,
      "silent_uninstall": false,
      "silent_upgrade": true,
      "auto_upgrade_from": null
    }
  ]
}
EOF
