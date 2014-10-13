# this is for testing
require 'optparse'
require 'faker'
require 'fileutils'

@max_black_count = 5000
@logs_count = 30000
@users_tsv = 'dummy_users.tsv'
@used_users_tsv = 'dummy_users_used.tsv'
@out_sql = 'dummy_log.sql'
opt = OptionParser.new(ARGV)
opt.on('--black-count CNT') {|x| @max_black_count = x.to_i }
opt.on('--logs-count CNT') {|x| @logs_count = x.to_i }
opt.on('--reserve-users N') {|x| @reserved_users = x.to_i }
opt.on('--users-tsv TSV') {|x| @users_tsv = x }
opt.on('--report-used-users-tsv TSV') {|x| @used_users_tsv = x }
opt.on('--output-sql SQL') {|x| @out_sql = x }
opt.parse!

@used_users = {}
@users = File.read(@users_tsv).each_line.map do |_|
  id, login, pass, salt, hash = _.chomp.split(/\t/)
  {id: id, login: login, pass: pass, salt: salt, hash: hash, fails: 0}
end

@reserved_users ||= @users.size/2

puts "users: #{@users_tsv}"
puts "output SQL: #{@out_sql}"
puts "output black users TSV: #{@used_users_tsv}"
puts ""
puts "Reserve #{@reserved_users} users, max #{@logs_count} logs, allows generating #{@max_black_count} locked users"

@users.shift(@reserved_users)


@black_count = 0

@next_ip = [127,200,1,1]

def get_ip
  ip = @next_ip.dup

  @next_ip[-1] += 1
  3.downto(0).each do |idx|
    if ip[idx] > 253
      @next_ip[idx-1] += 1
      @next_ip[idx] = 1
    end
  end
  raise ArgumentError if ip[0] > 127

  ip.join(".")
end

def get_safe_ip
  [127,250,0,(1..254).to_a.sample].join('.')
end

@login_log = []
@messages = []
@next_time = Time.new(2014,2,22,0,0,0)
def attempt(user, ip, succeeded, comment='')
  msg = "#{succeeded ? 'SUCCESS' : 'FAIL'}\t#{user.inspect}\t#{ip}\t#{comment}"
  puts msg if ENV['DEBUG']
  @messages << msg

  created_at = @next_time
  @next_time += rand(5).succ

  @used_users[user[:id]]=user
  if succeeded
    user[:fails] = 0
  else
    user[:fails] += 1
  end

  if 3 <= user[:fails]
    @black_count += 1
    @users.delete user
  end

  log = "('#{created_at}', #{user[:id]}, '#{user[:login]}', '#{ip}', #{succeeded ? 1 : 0})"
  @login_log << log
  log
end

(1..@logs_count).each do |i|
  user = @users.sample
  if @max_black_count <= @black_count
    succeeded = true
  else
    succeeded = [false, false, false, false, false, false, true, true, true, true].sample
  end
  attempt(user, succeeded ? get_safe_ip : get_ip, succeeded)
end

FileUtils.cp(@used_users_tsv,"#{@used_users_tsv}.old") if File.exist?(@used_users_tsv)
FileUtils.cp(@out_sql,"#{@out_sql}.old") if File.exist?(@out_sql)

open(@used_users_tsv, 'w') do |io|
  @used_users.each_value do |black|
    io.puts([black[:id], black[:login], black[:fails]].join("\t"))
  end
end

open(@out_sql, 'w') do |io|
  io.puts "-- Reserve #{@reserved_users} users, max #{@logs_count} logs"
  io.puts "-- Result: #{@black_count} users is locked, #{@used_users.size} users is used, and #{@login_log.size} logs consumed"

  @messages.each do |msg|
    io.puts "-- #{msg}"
  end

  io.puts

  @login_log.each_slice(10000) do |slice|
    io.puts "INSERT INTO `login_log` (`created_at`, `user_id`, `login`, `ip`, `succeeded`) VALUES #{slice.join(",")};"
  end
end

puts "==="
puts "Reserve #{@reserved_users} users, max #{@logs_count} logs"
puts "Result: #{@black_count} users is locked, #{@used_users.size} users is used, and #{@login_log.size} logs consumed"

