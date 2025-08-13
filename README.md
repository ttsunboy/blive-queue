原来那位佬随缘更新了，自己根据需求搓了一个改版出来。目前改动部分还是依托，后面慢慢优化。<del>bug和我有一个能跑就行</del>

- 从内存List改成了SQLite文件存储队列信息；
- 适配了大航海和礼物插队；
- 添加了一个Cookie输入框以稳定获得弹幕信息。万恶的婶婶！

排队规则：
1. 新购大航海优先，航海等级降序>时间升序
1. 已有大航海次之，航海等级降序>时间升序
1. 礼物≥52电池（情书），礼物价值降序>时间升序
1. 普通排队，时间升序

赶工所以规则是写死在代码里的，TODO：整一个自定义规则。

以下是原来的ReadMe：

---
---

<div align="center">

<img src="https://user-images.githubusercontent.com/36563862/171974383-fa4066b7-331e-4550-9d97-0b2e36791a4c.png" width="200" height="200" alt="排队姬">

# 排队姬
_✨ 简单快捷的b站直播排队插件！ ✨_

</div>

<p align="center">
  <a href="https://raw.githubusercontent.com/Akegarasu/blive-queue/master/LICENSE">
    <img src="https://img.shields.io/github/license/Akegarasu/blive-queue" alt="license">
  </a>
  <a href="https://github.com/Akegarasu/blive-queue/releases">
    <img src="https://img.shields.io/github/v/release/Akegarasu/blive-queue?color=blueviolet&include_prereleases" alt="release">
  </a>
  <a href="https://goreportcard.com/report/github.com/Akegarasu/blive-queue">
    <img src="https://goreportcard.com/badge/github.com/Akegarasu/blive-queue" alt="GoReportCard">
  </a>
</p>

<p align="center">
  <a href="https://github.com/Akegarasu/blive-queue/releases">下载</a>
  ·
  <a href="https://github.com/Akegarasu/blive-queue/blob/main/README.md">文档</a>
</p>

## 简介
blive-queue 是一个适用于obs、直播姬的 bilibili 直播弹幕排队插件~ 便捷使用方便配置，支持使用弹幕姬的CSS样式！开源、免费！

为主播解决一系列观众参加型活动、游戏等排队需求

## 功能

- 弹幕排队 发送关键词 “排队” 可加入排队队列

- 取消排队 发送关键词 “取消排队” 可以取消排队

- 完善的后台管理： 拖动排序、手动删除排队、一键清空排队~

- 支持牌子等级、大航海等级过滤 (牌子等级、舰长过滤)

- 支持使用弹幕姬样式，兼容大部分弹幕姬样式

- 便捷的使用方法：如果你使用弹幕姬那么排队姬和弹幕姬的操作几乎一模一样~ 没有使用过也可以下载并且几分钟内配制好

## 使用教程

b站专栏 [排队姬](https://www.bilibili.com/read/cv16545025)

## 支持作者

[秋葉的爱发电](https://afdian.net/@akibanzu)

## 前端代码

前端代码修改自 [blivechat](https://github.com/xfgryujk/blivechat)