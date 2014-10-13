require 'optparse'
require 'faker'
require 'digest/sha2'

$count = 20
opt = OptionParser.new(ARGV)
opt.on('--count count') {|x| $count = x.to_i }
opt.parse!


usernames = {}

sql = open(File.join(__dir__, 'dummy_users.sql'), 'w')
tsv = open(File.join(__dir__, 'dummy_users.tsv'), 'w')

sql.puts "INSERT INTO `users` (`id`, `login`, `password_hash`, `salt`) VALUES"

(1..$count).each do |i|
  if i < 10
    user = 'isucon%d' % i
    pass = 'isuconpass%d' % i
    salt = 'salt%d' % i
  else
    user = Faker::Internet.user_name
    while usernames[user]
      user = user.succ
    end

    pass = Faker::Internet.password
    salt = Faker::Internet.password
  end
  usernames[user] = true

  hash = Digest::SHA256.hexdigest("#{pass}:#{salt}")

  sql.print ',' if i > 1
  sql.puts "(#{i}, '#{user}', SHA2('#{pass}:#{salt}', 256), '#{salt}')"
  tsv.puts "#{i}\t#{user}\t#{pass}\t#{salt}\t#{hash}"
end

sql.close
tsv.close
