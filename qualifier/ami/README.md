# AMI の作り方

## インスタンスを立てる

事前に AWS\_ACCESS\_KEY\_ID と AWS\_SECRET\_ACCESS\_KEY 環境変数をセットしておくこと。

```sh
./pack.rb run_instance -s [security_group_name] -k [key_name]
```

このとき `tmp/instance_id` に instance_id が吐かれる。移行のコマンドはデフォルトではこれがインスタンスIDとして使われる。`-i` でも指定可能。

## セットアップ

```sh
./pack.rb provision
```
Ansible で benchmarker 以外がプロビジョンされる。

```sh
./pack.rb build_benchmarker
```
これで benchmarker がビルドされて `/home/isucon` に置かれる。

benchmarker のビルドは重いので `provision` と分けている。

## テスト

```sh
./pack.rb spec
```

Serverspec が走る。

## インスタンスに SSH で入りたいとき

```sh
./pack.rb ssh
```

## AMI を作る

```sh
./pack.rb create_image
```

## インスタンスを消す

```sh
./pack.rb terminate
```

`tmp/instance_id` も消える。

