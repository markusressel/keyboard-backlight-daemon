<h1 align="center">
  <br>
  keyboard-backlight-daemon
  <br>
</h1>

<h4 align="center">A daemon to make your keyboard backlight smart.</h4>

<div align="center">

[![Programming Language](https://img.shields.io/badge/Go-00ADD8?logo=go&logoColor=white)]()
[![Latest Release](https://img.shields.io/github/release/markusressel/keyboard-backlight-daemon.svg)](https://github.com/markusressel/keyboard-backlight-daemon/releases)
[![License](https://img.shields.io/badge/license-AGPLv3-blue.svg)](/LICENSE)

</div>

# Features

* [x] Light up keyboard backlight based on user interaction (keyboard, mouse, touchpad)
* [x] No system dependencies

# Tested on

* [x] Asus ROG Zephyrus M16 (2021)

# How to use

## Installation

### ![](https://img.shields.io/badge/Arch_Linux-1793D1?logo=arch-linux&logoColor=white)

```shell
yay -S keyboard-backlight-daemon-git
```

### Manual

Download the latest release from GitHub:

```shell
curl -L -o keyboard-backlight-daemon https://github.com/markusressel/keyboard-backlight-daemon/releases/latest/download/keyboard-backlight-daemon-linux-amd64
chmod +x keyboard-backlight-daemon
sudo cp ./keyboard-backlight-daemon /usr/bin/keyboard-backlight-daemon
sudo keyboard-backlight-daemon
```

Or compile yourself:

```shell
git clone https://github.com/markusressel/keyboard-backlight-daemon.git
cd keyboard-backlight-daemon
make build
sudo cp ./bin/keyboard-backlight-daemon /usr/bin/keyboard-backlight-daemon
sudo chmod ug+x /usr/bin/keyboard-backlight-daemon
sudo keyboard-backlight-daemon
```

## Configuration

If you want to change the default behaviour of keyboard-backlight-daemon you can create a YAML configuration file in **
one** of the following locations:

* `/etc/keyboard-backlight-daemon/keyboard-backlight-daemon.yaml` (recommended)
* `~/.config/keyboard-backlight-daemon.yaml`
* `./keyboard-backlight-daemon.yaml`

```shell
sudo mkdir /etc/keyboard-backlight-daemon
sudo nano /etc/keyboard-backlight-daemon/keyboard-backlight-daemon.yaml
```

## Run

### Systemd Service

Use the [systemd unit file](./keyboard-backlight-daemon.service) in this repository. To enable it simply run:

```shell
sudo cp ./keyboard-backlight-daemon.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now keyboard-backlight-daemon
# follow logs
journalctl -u keyboard-backlight-daemon -f
```

# Dependencies

See [go.mod](go.mod)

# Similar Projects

* [alexmohr/keyboard-backlight](https://github.com/alexmohr/keyboard-backlight)
* [ruben2020/kbd_backlight_ctrl](https://github.com/ruben2020/kbd_backlight_ctrl)
* [moson-mo/smartlight](https://github.com/moson-mo/smartlight)

# License

```
keyboard-backlight-daemon
Copyright (C) 2021  Markus Ressel

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published
by the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
```
