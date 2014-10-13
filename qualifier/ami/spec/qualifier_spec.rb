require 'spec_helper'

describe user('isucon') do
  it { should exist }
  it { should belong_to_group 'wheel' }
end

describe 'application files' do
  describe file('/home/isucon/init.sh') do
    it { should be_mode 755 }
    it { should be_owned_by 'isucon' }
  end

  describe file('/home/isucon/webapp') do
    it { should be_directory }
    it { should be_owned_by 'isucon' }
  end

  describe file('/home/isucon/sql') do
    it { should be_directory }
    it { should be_owned_by 'isucon' }
  end

  describe file('/home/isucon/webapp/ruby') do
    it { should be_directory }
    it { should be_owned_by 'isucon' }
  end
end

describe 'nginx' do
  describe file('/etc/nginx/nginx.conf') do
    it { should contain 'server 127.0.0.1:8080;' }
  end

  describe package('nginx') do
    it { should be_installed }
  end

  describe process('nginx') do
    it { should be_running }
  end

  describe command('curl -vs localhost') do
    its(:stdout) { should match /\b200 OK\b/ }
  end
end

describe 'supervisor' do
  describe process('supervisord') do
    it { should be_running }
  end

  describe command('supervisord -v') do
    its(:stdout) { should eq "3.1.1\n" }
  end

  describe file('/etc/supervisord.conf') do
    it { should contain 'foreman start' }
  end
end

describe 'mysql' do
  describe package('mysql-server') do
    it { should be_installed }
  end

  describe process('mysqld') do
    it { should be_running }
  end

  describe command('mysql -u root isu4_qualifier -e "SELECT COUNT(1) FROM users"') do
    its(:stdout) { should match /\b200000\b/ }
  end
end

describe 'xbuild' do
  describe command('/home/isucon/env.sh ruby -v') do
    its(:stdout) { should match /\b2\.1\.3/ }
  end
end

describe 'benchmarker' do
  describe file('/home/isucon/benchmarker') do
    it { should_not be_directory }
  end

  describe command('cd /home/isucon; /home/isucon/env.sh /home/isucon/benchmarker help') do
    its(:stdout) { should match /^USAGE:/ }
    its(:stdout) { should match /\bv2\b/ }
  end
end

describe file('/home/isucon/gocode/pkg') do
  it { should be_directory }
  it { should be_owned_by 'isucon' }
end

describe file('/etc/nginx/nginx.php.conf') do
  it { should contain 'mime.type' }
end
