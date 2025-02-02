---
title: Getting Started
description: Definition and description of sarabi
slug: getting-started
---

Its simple to get started! One of the great thing about sarabi is how easy it is to setup and start using. You can deploy your first app in minutes!

## Prerequisites
The only supported OS currently is Linux. Ubuntu, Kali, Fedora etc are supported. Make sure you have a VPS running linux with ability to use `ssh`. System requirements depends solely on your need and how resources intensive your application(s) are. But typically, 2GB Memory and 40GB disk size are more than enough to get started.
## Installation
You will need to install sarabi server on your VPS, it is responsible for processing application deployment and management requests. The server is a single Golang binary which uses `sqlite` to persist application data e.g environment properties, configurations etc. 

To control the server, you will also need to install the sarabi cli. Below are the steps.

1. Install sarabi server
```shell
curl https://github.com/adxgun/sarabi/install.sh | sh
```
2. Verify Installation

## Quickstart - Deploy your first application