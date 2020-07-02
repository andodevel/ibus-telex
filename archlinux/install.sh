#!/bin/bash
if [ -d ibus-telex ]; then
	echo "Tìm thấy thư mục tên ibus-telex, đổi tên thành ibus-telex-bak"
        mv ibus-telex ibus-telex-bak
fi

if [ -f ibus-telex ]; then
	echo "Tìm thấy file tên ibus-telex, đổi tên thành ibus-telex~"
        mv ibus-telex ibus-telex~
fi

echo "Chọn phiên bản muốn cài:"
echo "1. Bản git, mã nguồn mới nhất lấy từ github"
echo "2. Bản release"
echo "3. Thoát"
echo "Lựa chọn (1/2/3): "
read choice
case $choice in
	"1") VER="git";;
	"2") VER="release";;
	*) exit -1;;
esac

mkdir ibus-telex
cd ibus-telex
wget "https://raw.githubusercontent.com/andodevel/ibus-telex/master/archlinux/PKGBUILD-$VER" -O PKGBUILD
makepkg -si

cd ..
rm ibus-telex -rf
rm install.sh
