#!/usr/bin/env ruby
require 'thor'
require 'logger'
require 'pathname'
require 'json'

class CLI < Thor
  class_option :verbose, type: :boolean, default: false
  class_option :ssh_key, type: :string, default: File.expand_path('~/.ssh/google_compute_engine')

  desc 'run_instance', 'Run new instance'
  option :project, required: true
  option :name, required: true
  option :zone, default: 'asia-east1-b'
  option :machine_type, default: 'n1-standard-4'
  option :network, default: 'default'
  option :maintenance_policy, default: 'TERMINATE'
  option :scopes, default: ['https://www.googleapis.com/auth/devstorage.read_write', 'https://www.googleapis.com/auth/logging.write']
  option :tags, default: ['http-server']
  option :image, default: 'https://www.googleapis.com/compute/v1/projects/centos-cloud/global/images/centos-6-v20150710'
  option :boot_disk_size, default: 10
  option :boot_disk_type, default: 'pd-standard'
  option :boot_disk_device_name, default: 'image0'
  def run_instance
    args = options.merge({
      no_restart_on_failure: nil,
      no_boot_disk_auto_delete: nil,
    })
    project = args.delete(:project)
    name = args.delete(:name)

    args.delete(:verbose)
    args.delete(:ssh_key)

    instance = create_instance(project: project, name: name, args: args)
    instance_info_path.open('w') {|f| f.write instance.to_json }
    say_status 'run_instance', instance["name"]
  end

  desc 'provision', 'Provision the instance using ansible'
  def provision
    playbooks = Dir.glob('ansible/*.yml').reject{|x| x=~%r!/_[^/]+\.yml$! }.sort
    run_playbooks playbooks
  end

  desc 'build_benchmarker', 'Build benchmarker'
  def build_benchmarker
    warn "Run `pack.rb provision` to provision prerequisites"
    run_playbooks 'ansible/05_benchmarker.yml'
  end

  desc 'cleanup', 'Delete log files from the instance'
  def cleanup
    run_playbooks 'ansible/_cleanup.yml'
  end

  desc 'spec', 'Test the instance using serverspec'
  def spec
    # ssh -i KEY_FILE -o UserKnownHostsFile=/dev/null -o CheckHostIP=no -o StrictHostKeyChecking=no USER@IP_ADDRESS
    #  ~/.ssh/google_compute_engine
    say_status 'run', "TARGET_HOST=#{public_ip_address} PRIVATE_KEY_PATH=#{options[:ssh_key]} bundle exec rspec"
    system({'TARGET_HOST' => public_ip_address, 'PRIVATE_KEY_PATH' => options[:ssh_key]}, *%w(bundle exec rspec))
    exit $?.exitstatus || 254
  end

  desc 'start', 'Start the instance'
  def start
    raise NotImplementedError
    instance = instance_info
    ec2.instances[id].start
    say_status 'start', id
  end

  desc 'stop', 'Stop the instance'
  def stop
    raise NotImplementedError
    id = instance_id
    ec2.instances[id].stop
    say_status 'stop', id
  end

  desc 'terminate', 'Terminate the instance'
  def terminate
    raise NotImplementedError
    id = instance_id
    ec2.instances[id].terminate
    instance_id_path.delete
    say_status 'terminate', id
  end

  desc 'create_image', 'Create AMI from the instance'
  def create_image
    raise NotImplementedError
    # Do it by yourself...
    # https://cloud.google.com/compute/docs/images#export_an_image_to_google_cloud_storage

    # gcloud compute disks create temporary-disk --zone ZONE
    # gcloud compute instances attach-disk example-instance --disk temporary-disk \
    #     --device-name temporary-disk \
    #     --zone ZONE
    # gcloud compute ssh example-instance
    # $ sudo mkdir /mnt/tmp
    # $ sudo /usr/share/google/safe_format_and_mount -m "mkfs.ext4 -F" /dev/sdb /mnt/tmp
    # $ sudo gcimagebundle -d /dev/sda -o /mnt/tmp/ --log_file=/tmp/abc.log
    # $ gsutil mb gs://BUCKET_NAM
    # $ gsutil cp /mnt/tmp/IMAGE_NAME.image.tar.gz gs://BUCKET_NAME
  end

  desc 'ssh', 'Login the instance via ssh'
  def ssh
    # gcloud compute --project "isucon5-summer-course" ssh --zone "asia-east1-b" "image-test"
    instance = instance_info
    command = ['gcloud', 'compute', '--project', instance['project'], 'ssh', '--zone', instance['zone'], instance['name']].join(' ')
    say_status 'exec', command
    exec command
  end

  no_tasks do
    def tmpdir
      dir = Pathname('../tmp').expand_path(__FILE__)
      dir.mkdir unless dir.directory?
      dir
    end

    def instance_info_path
      tmpdir.join('instance_info')
    end

    def instance_info
      @instance_info ||= begin
        JSON.parse(instance_info_path.read)
      end
    rescue Errno::ENOENT
      abort "#{instance_info_path} doesn't exist"
    end

    def create_instance(project:, name:, args: {})
      cmd = ['gcloud', '--format', 'json', 'compute', '--project', project, 'instances', 'create', name]
      cmd += build_cli_options(args)

      io = IO.popen(cmd)
      instance = JSON.parse(io.read).first rescue nil
      unless instance
        raise "failed to create instance"
      end

      example = {
        "canIpForward": false,
        "cpuPlatform": "Intel Ivy Bridge",
        "creationTimestamp": "2015-08-14T22:59:02.111-07:00",
        "disks": [
          { "boot": true, "deviceName": "image0", "index": 0, "interface": "SCSI", "kind": "compute#attachedDisk",
            "mode": "READ_WRITE", "source": "image-test4", "type": "PERSISTENT" }
        ],
        "id": "17839858961528212297",
        "kind": "compute#instance",
        "machineType": "n1-standard-4",
        "metadata": {"fingerprint": "3d2QLXihB6g=", "kind": "compute#metadata"},
        "name": "image-test4",
        "networkInterfaces": [
          { "accessConfigs": [
              { "kind": "compute#accessConfig", "name": "external-nat", "natIP": "130.211.253.182", "type": "ONE_TO_ONE_NAT" }
            ],
            "name": "nic0",
            "network": "default",
            "networkIP": "10.240.164.226"
          }
        ],
        "scheduling": { "automaticRestart": false, "onHostMaintenance": "TERMINATE", "preemptible": false },
        "selfLink": "https://www.googleapis.com/compute/v1/projects/isucon5-summer-course/zones/asia-east1-b/instances/image-test4",
        "serviceAccounts": [
          {
            "email": "338645772293-compute@developer.gserviceaccount.com",
            "scopes": [
              "https://www.googleapis.com/auth/devstorage.read_only",
              "https://www.googleapis.com/auth/logging.write"
            ]
          }
        ],
        "status": "RUNNING",
        "tags": {
          "fingerprint": "FYLDgkTKlA4=",
          "items": [
            "http-server"
          ]
        },
        "zone": "asia-east1-b"
      }

      create_http_firewall_rule(project: project, name: instance["networkInterfaces"].first["name"], target_tag: instance["tags"]["items"].first)

      instance.merge({'project' => project})
    end

    def create_http_firewall_rule(project:, name:, target_tag:)
      list = ['gcloud', '--format', 'json', 'compute', '--project', project, 'firewall-rules', 'list']
      JSON.parse(IO.popen(list).read).each do |rule|
        return if rule["name"] == name
      end

      cmd = ['gcloud', '--format', 'json', 'compute', '--project', project, 'firewall-rules', 'create', name]
      cmd += ['--allow', 'tcp:80', '--network', 'default', '--source-ranges', '0.0.0.0/0', '--target-tags', target_tag]

      JSON.parse(IO.popen(cmd).read)
    end

    def build_cli_options(args)
      options = []
      args.each do |key, value|
        options << '--' + key.to_s.gsub(/_/, '-')
        if value
          if value.is_a? Array
            options << value.map(&:to_s).join(',')
          else
            options << value.to_s
          end
        end
      end
      options
    end

    def public_ip_address
      @public_ip_address ||= begin
        instance_info['networkInterfaces'][0]["accessConfigs"][0]["natIP"]
      end
    end

    def run_playbooks(playbooks)
      # ssh -i KEY_FILE -o UserKnownHostsFile=/dev/null -o CheckHostIP=no -o StrictHostKeyChecking=no USER@IP_ADDRESS
      #  ~/.ssh/google_compute_engine
      playbooks = [playbooks] unless playbooks.is_a?(Array)
      opts = "-i '#{public_ip_address},'"
      opts += " --private-key=#{options[:ssh_key]}"
      opts += " --verbose -vvvv" if options[:verbose]
      command = "ansible-playbook #{opts} #{playbooks.join(' ')}"
      say_status 'run', command
      system({'ANSIBLE_HOST_KEY_CHECKING' => 'False'}, command)
    end
  end
end

CLI.start
