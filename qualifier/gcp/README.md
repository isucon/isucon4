# AMI の作り方

事前に Google Cloud SDK (`gcloud`) をインストールしておくこと https://cloud.google.com/sdk/

## インスタンスを立てる


```sh
./pack.rb run_instance --project PROJECT --name NAME
```

このとき `tmp/instance_info` にいろいろ吐かれる。以降のコマンドはデフォルトでこれが作業対象インスタンスの情報として使われる。

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

## イメージを作る

TBD
