var _ = require('underscore');
var async = require('async');
var bodyParser = require('body-parser');
var crypto = require('crypto');
var ect = require('ect');
var express = require('express');
var logger = require('morgan');
var mysql = require('mysql');
var path = require('path');
var session = require('express-session');
var strftime = require('strftime');

var app = express();

var globalConfig = {
  userLockThreshold: process.env.ISU4_USER_LOCK_THRESHOLD || 3,
  ipBanThreshold: process.env.ISU4_IP_BAN_THRESHOLD || 10
};

var mysqlPool = mysql.createPool({
  host: process.env.ISU4_DB_HOST || 'localhost',
  user: process.env.ISU4_DB_USER || 'root',
  password: process.env.ISU4_DB_PASSWORD || '',
  database: process.env.ISU4_DB_NAME || 'isu4_qualifier'
});

var helpers = {
  calculatePasswordHash: function(password, salt) {
    var c = crypto.createHash('sha256');
    c.update(password + ':' + salt);
    return c.digest('hex');
  },

  isUserLocked: function(user, callback) {
    if(!user) {
      return callback(false);
    };

    mysqlPool.query(
      'SELECT COUNT(1) AS failures FROM login_log WHERE ' +
      'user_id = ? AND id > IFNULL((select id from login_log where ' +
      'user_id = ? AND succeeded = 1 ORDER BY id DESC LIMIT 1), 0);',
      [user.id, user.id],
      function(err, rows) {
        if(err) {
          return callback(false);
        }

        callback(globalConfig.userLockThreshold <= rows[0].failures);
      }
    )
  },

  isIPBanned: function(ip, callback) {
    mysqlPool.query(
      'SELECT COUNT(1) AS failures FROM login_log WHERE ' +
      'ip = ? AND id > IFNULL((select id from login_log where ip = ? AND ' +
      'succeeded = 1 ORDER BY id DESC LIMIT 1), 0);',
      [ip, ip],
      function(err, rows) {
        if(err) {
          return callback(false);
        }

        callback(globalConfig.ipBanThreshold <= rows[0].failures);
      }
    )
  },

  attemptLogin: function(req, callback) {
    var ip = req.headers['x-forwarded-for'] || req.connection.remoteAddress;
    var login = req.body.login;
    var password = req.body.password;

    async.waterfall([
      function(cb) {
        mysqlPool.query('SELECT * FROM users WHERE login = ?', [login], function(err, rows) {
          cb(null, rows[0]);
        });
      },
      function(user, cb) {
        helpers.isIPBanned(ip, function(banned) {
          if(banned) {
            cb('banned', user);
          } else {
            cb(null, user);
          };
        });
      },
      function(user, cb) {
        helpers.isUserLocked(user, function(locked) {
          if(locked) {
            cb('locked', user);
          } else {
            cb(null, user);
          };
        });
      },
      function(user, cb) {
        if(user && helpers.calculatePasswordHash(password, user.salt) == user.password_hash) {
          cb(null, user);
        } else if(user) {
          cb('wrong_password', user);
        } else {
          cb('wrong_login', user);
        };
      }
    ], function(err, user) {
      var succeeded = !err;
      mysqlPool.query(
        'INSERT INTO login_log' +
        ' (`created_at`, `user_id`, `login`, `ip`, `succeeded`)' +
        ' VALUES (?,?,?,?,?)',
        [new Date(), (user || {})['id'], login, ip, succeeded],
        function(e, rows) {
          callback(err, user);
        }
      );
    });
  },

  getCurrentUser: function(user_id, callback) {
    mysqlPool.query('SELECT * FROM users WHERE id = ?', [user_id], function(err, rows) {
      if(err) {
        return callback(null);
      }

      callback(rows[0]);
    });
  },

  getBannedIPs: function(callback) {
    mysqlPool.query(
      'SELECT ip FROM (SELECT ip, MAX(succeeded) as max_succeeded, COUNT(1) as cnt FROM '+
      'login_log GROUP BY ip) AS t0 WHERE t0.max_succeeded = 0 AND t0.cnt >= ?',
      [globalConfig.ipBanThreshold],
      function(err, rows) {
        var bannedIps = _.map(rows, function(row) { return row.ip; });

        mysqlPool.query(
          'SELECT ip, MAX(id) AS last_login_id FROM login_log WHERE succeeded = 1 GROUP by ip',
          function(err, rows) {
            async.parallel(
              _.map(rows, function(row) {
                return function(cb) {
                  mysqlPool.query(
                    'SELECT COUNT(1) AS cnt FROM login_log WHERE ip = ? AND ? < id',
                    [row.ip, row.last_login_id],
                    function(err, rows) {
                      if(globalConfig.ipBanThreshold <= (rows[0] || {})['cnt']) {
                        bannedIps.push(row['ip']);
                      }
                      cb(null);
                    }
                  );
                };
              }),
              function(err) {
                callback(bannedIps);
              }
            );
          }
        );
      }
    )
  },

  getLockedUsers: function(callback) {
    mysqlPool.query(
      'SELECT user_id, login FROM ' +
      '(SELECT user_id, login, MAX(succeeded) as max_succeeded, COUNT(1) as cnt FROM ' +
      'login_log GROUP BY user_id) AS t0 WHERE t0.user_id IS NOT NULL AND ' +
      't0.max_succeeded = 0 AND t0.cnt >= ?',
      [globalConfig.userLockThreshold],
      function(err, rows) {
        var lockedUsers = _.map(rows, function(row) { return row['login']; });

        mysqlPool.query(
          'SELECT user_id, login, MAX(id) AS last_login_id FROM login_log WHERE ' +
          'user_id IS NOT NULL AND succeeded = 1 GROUP BY user_id',
          function(err, rows) {
            async.parallel(
              _.map(rows, function(row) {
                return function(cb) {
                  mysqlPool.query(
                    'SELECT COUNT(1) AS cnt FROM login_log WHERE user_id = ? AND ? < id',
                    [row['user_id'], row['last_login_id']],
                    function(err, rows) {
                      if(globalConfig.userLockThreshold <= (rows[0] || {})['cnt']) {
                        lockedUsers.push(row['login']);
                      };
                      cb(null);
                    }
                  );
                };
              }),
              function(err) {
                callback(lockedUsers);
              }
            );
          }
        );
      }
    )
  }
};

app.use(logger('dev'));
app.enable('trust proxy');
app.engine('ect', ect({ watch: true, root: __dirname + '/views', ext: '.ect' }).render);
app.set('view engine', 'ect');
app.use(bodyParser.urlencoded({ extended: false }));
app.use(session({ 'secret': 'isucon4-node-qualifier', resave: true, saveUninitialized: true }));
app.use(express.static(path.join(__dirname, '../public')));

app.locals.strftime = function(format, date) {
  return strftime(format, date);
};

app.get('/', function(req, res) {
  var notice = req.session.notice;
  req.session.notice = null;

  res.render('index', { 'notice': notice });
});

app.post('/login', function(req, res) {
  helpers.attemptLogin(req, function(err, user) {
    if(err) {
      switch(err) {
        case 'locked':
          req.session.notice = 'This account is locked.';
          break;
        case 'banned':
          req.session.notice = "You're banned.";
          break;
        default:
          req.session.notice = 'Wrong username or password';
          break;
      }

      return res.redirect('/');
    }

    req.session.userId = user.id;
    res.redirect('/mypage');
  });
});

app.get('/mypage', function(req, res) {
  helpers.getCurrentUser(req.session.userId, function(user) {
    if(!user) {
      req.session.notice = "You must be logged in"
      return res.redirect('/')
    }

    mysqlPool.query(
      'SELECT * FROM login_log WHERE succeeded = 1 AND user_id = ? ORDER BY id DESC LIMIT 2',
      [user.id],
      function(err, rows) {
        var lastLogin = rows[rows.length-1];
        res.render('mypage', { 'last_login': lastLogin });
      }
    );
  });
});

app.get('/report', function(req, res) {
  async.parallel({
    banned_ips: function(cb) {
      helpers.getBannedIPs(function(ips) {
        cb(null, ips);
      });
    },
    locked_users: function(cb) {
      helpers.getLockedUsers(function(users) {
        cb(null, users);
      });
    }
  }, function(err, result) {
    res.json(result);
  });
});

app.use(function (err, req, res, next) {
  res.status(500).send('Error: ' + err.message);
});

var server = app.listen(process.env.PORT || 8080, function() {
  console.log('Listening on port %d', server.address().port);
});
