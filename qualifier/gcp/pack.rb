#!/usr/bin/env ruby
require 'thor'
require 'aws-sdk'
require 'logger'
require 'pathname'

class CLI < Thor
  class_option :instance_id, aliases: :i
  class_option :verbose, type: :boolean, default: false

  desc 'run_instance', 'Run new instance'
  option :base_image_id, aliases: :b, default: 'ami-29dc9228' # hvm amazon-linux
  option :key_name, aliases: :k, required: true
  option :instance_type, aliases: :t, default: 'm3.xlarge'
  option :security_groups, aliases: :s, default: %w[default]
  option :init_file
  def run_instance
    user_data = ''
    user_data = File.open(options[:init_file]).read if options[:init_file]
    args = {
      image_id: options[:base_image_id],
      instance_type: options[:instance_type],
      key_name: options[:key_name],
      security_groups: options[:security_groups],
      user_data: user_data
    }
    instance = ec2.instances.create args
    instance_id_path.open('w') {|f| f.write instance.id }
    instance.tags['Name'] = "isucon4-pack-#{Time.now.to_i}"
    say_status 'run_instance', instance.id
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
    command = "TARGET_HOST=#{public_ip_address} bundle exec rspec"
    say_status 'run', command
    system command
  end

  desc 'start', 'Start the instance'
  def start
    id = instance_id
    ec2.instances[id].start
    say_status 'start', id
  end

  desc 'stop', 'Stop the instance'
  def stop
    id = instance_id
    ec2.instances[id].stop
    say_status 'stop', id
  end

  desc 'terminate', 'Terminate the instance'
  def terminate
    id = instance_id
    ec2.instances[id].terminate
    instance_id_path.delete
    say_status 'terminate', id
  end

  desc 'create_image', 'Create AMI from the instance'
  def create_image
    now = Time.now
    name = "2014-qualifier-#{now.to_i}"
    desc = "Built #{now.to_s}"
    image = ec2.instances[instance_id].create_image(name, description: desc, no_reboot: true)
    say_status 'create_image', image.id
  end

  desc 'ssh', 'Login the instance via ssh'
  option :user, aliases: :u, default: 'ec2-user'
  def ssh
    command = "ssh #{options[:user]}@#{public_ip_address}"
    say_status 'exec', command
    exec command
  end

  no_tasks do
    def tmpdir
      dir = Pathname('../tmp').expand_path(__FILE__)
      dir.mkdir unless dir.directory?
      dir
    end

    def instance_id_path
      tmpdir.join('instance_id')
    end

    def instance_id
      @instance_id ||= begin
        id = options[:instance_id] || instance_id_path.read.chomp
        if id !~ /^i-\w+$/
          abort "Invalid instance id: '#{id}'"
        end
        id
      end
    rescue Errno::ENOENT
      abort "#{instance_id_path} doesn't exist"
    end

    def ec2
      @ec2 ||= begin
        logger = options[:verbose] ? Logger.new($stdout).tap{|l| l.level = Logger::DEBUG } : nil
        AWS::EC2.new(
          access_key_id: ENV['AWS_ACCESS_KEY_ID'],
          secret_access_key: ENV['AWS_SECRET_ACCESS_KEY'],
          logger: logger
        )
      end
    end

    def wait_for_start
      loop do
        id = instance_id
        result = ec2.client.describe_instance_status(instance_ids: [id])
        status_set = result[:instance_status_set].first
        instance_status = status_set[:instance_status][:status] rescue nil
        system_status = status_set[:system_status][:status] rescue nil
        say_status 'checking', "status(#{id}) #{system_status} / #{instance_status}"
        break if instance_status == 'ok' && system_status == 'ok'
        sleep 3
      end
    end

    def public_ip_address
      @public_ip_address ||= begin
        wait_for_start
        instance = ec2.instances[instance_id]
        instance.public_ip_address
      end
    end

    def run_playbooks(playbooks)
      playbooks = [playbooks] unless playbooks.is_a?(Array)
      opts = "-i '#{public_ip_address},'"
      opts += " --verbose" if options[:verbose]
      command = "ansible-playbook #{opts} #{playbooks.join(' ')}"
      say_status 'run', command
      system command
    end
  end
end

CLI.start
