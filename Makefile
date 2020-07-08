#
# Telex - A Vietnamese Input method editor
# Copyright (C) 2018 Luong Thanh Lam <ltlam93@gmail.com>
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with this program.  If not, see <http://www.gnu.org/licenses/>.
#

SHELL := /bin/bash

# FIXME: Variable not work at all
VERSION := $(SHELL cat ./VERSION).
LDFLAGS += -X "main.BuildTimestamp=$(SHELL date -u "+%Y-%m-%d %H:%M:%S")"
LDFLAGS += -X "main.Version=$(VERSION)$(SHELL git rev-parse --short HEAD)"

GO := GO111MODULE=on go

# Developement
.PHONY: dev-tools
dev-tools:
	go get -u golang.org/x/lint/golint
	go get -u golang.org/x/tools/cmd/goimports

# Initialize project. All init(s) target should be run only once to create the project.
.PHONY: init
init: dev-tools
	go get -u github.com/BambooEngine/goibus/ibus
	go get -u github.com/godbus/dbus
	@echo "Install pre-commit hooks"
	@chmod +x ./hack/check.sh
	@chmod +x ./hooks/pre-commit
	@ln -sf ./hooks/pre-commit ./.git/hooks/pre-commit || true

engine_name=telex
ibus_e_name=ibus-engine-$(engine_name)
pkg_name=ibus-$(engine_name)
version=0.0.1

engine_dir=/usr/share/$(pkg_name)
ibus_dir=/usr/share/ibus

rpm_src_dir=~/rpmbuild/SOURCES
tar_file=$(pkg_name)-$(version).tar.gz
rpm_src_tar=$(rpm_src_dir)/$(tar_file)
tar_options_src=--transform "s/^\./$(pkg_name)-$(version)/" --exclude={"*.tar.gz",".git",".idea"} .

build:
	@$(GO) build -ldflags="-s -w" -o $(ibus_e_name) ./src/engine

clean:
	rm -f ibus-engine-* *_linux *_cover.html go_test_* go_build_* test *.gz test
	rm -f debian/files
	rm -rf debian/debhelper*
	rm -rf debian/.debhelper
	rm -rf debian/ibus-telex*


install: build
	mkdir -p $(DESTDIR)$(engine_dir)
	mkdir -p $(DESTDIR)/usr/lib/
	mkdir -p $(DESTDIR)$(ibus_dir)/component/

	cp -R -f ibus-telex.png $(DESTDIR)$(engine_dir)
	cp -f $(ibus_e_name) $(DESTDIR)/usr/lib/
	cp -f $(engine_name).xml $(DESTDIR)$(ibus_dir)/component/


uninstall:
	sudo rm -rf $(DESTDIR)$(engine_dir)
	sudo rm -f $(DESTDIR)/usr/lib/$(ibus_e_name)
	sudo rm -f $(DESTDIR)$(ibus_dir)/component/$(engine_name).xml


src: clean
	tar -zcf $(DESTDIR)/$(tar_file) $(tar_options_src)
	cp -f $(pkg_name).spec $(DESTDIR)/
	cp -f $(pkg_name).dsc $(DESTDIR)/
	cp -f debian/changelog $(DESTDIR)/debian.changelog
	cp -f debian/control $(DESTDIR)/debian.control
	cp -f debian/rules $(DESTDIR)/debian.rules
	cp -f archlinux/PKGBUILD-release $(DESTDIR)/PKGBUILD


rpm: clean
	tar -zcf $(rpm_src_tar) $(tar_options_src)
	rpmbuild $(pkg_name).spec -ba

deb: clean
	dpkg-buildpackage -uc -us


.PHONY: build clean build install uninstall src rpm deb
