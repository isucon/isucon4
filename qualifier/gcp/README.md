# ISUCON4 qualifier for GCE

事前に Google Cloud SDK (`gcloud`) をインストールしておくこと https://cloud.google.com/sdk/

## Images (built by @sorah)

- `gs://shirokanezoo/isucon4/qualifier/v3.tar.gz`
  - No warranty and @sorah doesn't provide any supports.

## Image creation

### インスタンスを立てる


```sh
./pack.rb run_instance --project PROJECT --name NAME
```

このとき `tmp/instance_info` にいろいろ吐かれる。以降のコマンドはデフォルトでこれが作業対象インスタンスの情報として使われる。

### セットアップ

```sh
./pack.rb provision
```

Ansible で benchmarker 以外がプロビジョンされる。

```sh
./pack.rb build_benchmarker
```

これで benchmarker がビルドされて `/home/isucon` に置かれる。

benchmarker のビルドは重いので `provision` と分けている。

### テスト

```sh
./pack.rb spec
```

Serverspec が走る。

### インスタンスに SSH で入りたいとき

```sh
./pack.rb ssh
```

### イメージを作る

- https://cloud.google.com/compute/docs/images#export_an_image_to_google_cloud_storage

```
$ export CLOUDSDK_CORE_PROJECT=PROJECT

$ gcloud compute disks create $TEMPORARY_DISK --zone $ZONE

$ gcloud compute instances attach-disk $YOUR_INSTANCE \
$ --disk $TEMPORARY_DISK \
$ --device-name temporary-disk \
$ --zone $ZONE

$ gcloud compute ssh $YOUR_INSTANCE
remote $ rm ~/.bash_history
remote $ sudo mkdir /mnt/tmp
remote $ sudo /usr/share/google/safe_format_and_mount -m "mkfs.ext4 -F" /dev/disk/by-id/google-temporary-disk /mnt/tmp
remote $ sudo gcimagebundle -d /dev/sda -o /mnt/tmp/ --log_file=/tmp/gcimagebundle.log
remote $ gsutil cp /mnt/tmp/*.image.tar.gz gs://BUCKET_NAME/path/to/image.tar.gz

```
