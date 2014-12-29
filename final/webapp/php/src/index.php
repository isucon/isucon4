<?php
require_once 'limonade/lib/limonade.php';

function configure() {
  option('base_uri', '/');

  $redis = new Redis();
  $redis->connect('127.0.0.1', 6379);
  option('redis', $redis);

  $config = [
  ];
  option('config', $config);
}

function log_path($advertiser_id) {
  $dir = get_dir('log');
  $exploded = explode('/', $advertiser_id);
  $path = $dir . '/' . end($exploded);
  return $path;
}

function fetch($hash, $key, $default_value) {
  if (isset($hash[$key])) {
    return $hash[$key];
  }
  else {
    return $default_value;
  }
}

function incr_hash(&$hash, $key) {
  if (is_object($hash)) {
    $hash = [];
  }
  if (isset($hash[$key])) {
    $hash[$key] += 1;
  }
  else {
    $hash[$key] = 1;
  }
  return $hash[$key];
}

function url($path) {
  return 'http://' . $_SERVER['HTTP_HOST'] . $path;
}

// redirect_to in Limonade is broken...
function redirect_to_alt($url) {
  status(302);
  send_header('Location: '. $url);
  return '';
}

function content_type($type) {
  send_header('Content-Type: ' . $type);
}

function get_dir($name) {
  $base_dir = '/tmp/php/';
  $path = $base_dir . $name;
  mkdir($path, 0755, true);
  return $path;
}

function advertiser_id() {
  if (isset($_SERVER['HTTP_X_ADVERTISER_ID'])){
    return $_SERVER['HTTP_X_ADVERTISER_ID'];
  }
  else {
    return null;
  }
}

function ad_key($slot, $id) {
  return 'isu4:ad:' . $slot . '-' . $id;
}

function asset_key($slot, $id) {
  return 'isu4:asset:' . $slot . '-' . $id;
}

function advertiser_key($id) {
  return 'isu4:advertiser:' . $id;
}

function slot_key($slot) {
  return 'isu4:slot:' . $slot;
}

function next_ad_id() {
  $redis = option('redis');
  return $redis->incr('isu4:ad-next');
}

function next_ad($slot) {
  $redis = option('redis');

  $key = slot_key($slot);

  $id = $redis->rpoplpush($key, $key);
  if (empty($id)) {
    return null;
  }

  $ad = get_ad($slot, $id);
  if (!empty($ad)) {
    return $ad;
  }
  else {
    $redis->lrem($key, 0, $id);
    next_ad();
  }
}

function get_ad($slot, $id) {
  $redis = option('redis');

  $key = ad_key($slot, $id);
  $ad = $redis->hgetall($key);

  if (empty($ad)) {
    return null;
  }

  if (isset($ad['impressions'])) {
    $ad['impressions'] = 0;
  }
  $ad['asset']    = url('/slots/' . $slot . '/ads/' . $id . '/asset');
  $ad['counter']  = url('/slots/' . $slot . '/ads/' . $id . '/count');
  $ad['redirect'] = url('/slots/' . $slot . '/ads/' . $id . '/redirect');

  return $ad;
}

function decode_user_key($id) {
  if (empty($id)) {
    return ['gender' => 'unknown', 'age' => ''];
  }
  $arr = explode('/', $id);
  if ($arr[0] == 0) {
    $gender = 'female';
  }
  else {
    $gender = 'male';
  }
  $age = intval($arr[1]);
  return ['gender' => $gender, 'age' => $age];
}

function get_log($id) {
  $path = log_path($id);
  if (!file_exists($path)) {
    return [];
  }

  $result = [];
  $fp = fopen($path, 'r');
  if (!flock($fp, LOCK_SH)) {
    throw new RuntimeException('Cannot flock ' . $path);
  }
  while (!feof($fp)) {
    $line = fgets($fp);
    if (!$line) {
      break;
    }
    $line = rtrim($line);
    $cols = explode("\t", $line);
    $ad_id = $cols[0];
    $user  = $cols[1];
    $agent = $cols[2];
    $user_attr = decode_user_key($user);
    $result[$ad_id][]= [
      'ad_id' => $ad_id,
      'user'  => $user,
      'agent' => $agent,
      'age'   => $user_attr['age'],
      'gender'=> $user_attr['gender']
    ];
  }
  fclose($fp);
  return $result;
}

dispatch_get('/', function() {
  return render_file(option('public_dir').'/index.html');
});

dispatch_get('/index.html', function() {
  return render_file(option('public_dir').'/index.html');
});

dispatch_get('/view.html', function() {
  return render_file(option('public_dir').'/view.html');
});

dispatch_post('/slots/:slot/ads', function() {
  $advertiser_id = advertiser_id();

  if (empty($advertiser_id)) {
    return halt(400);
  }

  $slot = params('slot');
  $asset = $_FILES['asset'];

  $dir = get_dir('upload');
  $tmp_path = $dir . sprintf('/upload-%s', sha1_file($asset['tmp_name']));
  if (!move_uploaded_file($asset['tmp_name'], $tmp_path)) {
    throw new RuntimeException('Failed to move uploaded file.');
  }

  $id = next_ad_id();
  $key = ad_key($slot, $id);

  $redis = option('redis');
  $type = isset($_POST['type']) ? $_POST['type'] : ($asset['type'] ?: 'video/mp4');
  $redis->hmset($key, [
    'slot' => $slot,
    'id' => $id,
    'title' => $_POST['title'],
    'type' => $type,
    'advertiser' => $advertiser_id,
    'destination' => $_POST['destination'],
    'impressions' => 0
  ]);

  $redis->set(asset_key($slot, $id), file_get_contents($tmp_path));
  $redis->rpush(slot_key($slot), $id);
  $redis->sadd(advertiser_key($advertiser_id), $key);

  return json(get_ad($slot, $id));
});

dispatch_get('/slots/:slot/ad', function() {
  $slot = params('slot');
  $ad = next_ad($slot);
  if (!empty($ad)) {
    return redirect_to('/slots/' . $slot . '/ads/' . $ad['id']);
  }
  else {
    status(404);
    return json(['error' => 'not_found']);
  }
});

dispatch_get('/slots/:slot/ads/:id', function() {
  $ad = get_ad(params('slot'), params('id'));
  if (!empty($ad)) {
    return json($ad);
  }
  else {
    status(404);
    return json(['error' => 'not_found']);
  }
});

dispatch_get('/slots/:slot/ads/:id/asset', function() {
  $slot = params('slot');
  $id = params('id');
  $ad = get_ad($slot, $id);
  if (!empty($ad)) {
    $content_type = $ad['type'] ?: 'application/octet-stream';
    content_type($content_type);
    $redis = option('redis');
    $data = $redis->get(asset_key($slot, $id));

    if (isset($_SERVER['HTTP_RANGE'])) {
      $range = $_SERVER['HTTP_RANGE'];
    }
    else {
      send_header('Content-Length: ' . strlen($data));
      return $data;
    }

    if (preg_match('/\Abytes=(\d*)-(\d*)\z/', $range, $match)) {
      $head = $match[1];
      $tail = $match[2];
    }
    else {
      return halt(416);
    }
    if ($head == '' && $tail == '') {
      return halt(416);
    }
    $orig_size = strlen($data);
    $head = $head ?: 0;
    $tail = $tail ?: $orig_size - 1;
    if ($head < 0 || $head >= $orig_size || $tail < 0) {
      return halt(416);
    }
    $data = substr($data, $head, $tail - $head + 1);
    status(206);
    send_header('Content-Range: bytes ' . $head . '-' . $tail . '/' . $orig_size);
    send_header('Content-Length: ' . strlen($data));

    return $data;
  }
  else {
    status(404);
    return json(['error' => 'not_found']);
  }
});

dispatch_post('/slots/:slot/ads/:id/count', function() {
  $slot = params('slot');
  $id = params('id');
  $key = ad_key($slot, $id);

  $redis = option('redis');
  if (!$redis->exists($key)) {
    status(404);
    return json(['error' => 'not_found']);
  }

  $redis->hincrby($key, 'impressions', 1);
  return status(204);
});

dispatch_get('/slots/:slot/ads/:id/redirect', function() {
  $slot = params('slot');
  $id = params('id');
  $ad = get_ad($slot, $id);

  if (empty($ad)) {
    status(404);
    return json(['error' => 'not_found']);
  }

  $path = log_path($ad['advertiser']);
  $fp = fopen($path, 'a');
  if (!flock($fp, LOCK_EX)) {
    throw new RuntimeException('Cannot flock ' . $path);
  }
  if (isset($_COOKIE['isuad'])) {
    $isuad = $_COOKIE['isuad'];
  }
  else {
    $isuad = '';
  }
  if (isset($_SERVER['HTTP_USER_AGENT'])) {
    $ua = $_SERVER['HTTP_USER_AGENT'];
  }
  else {
    $ua = 'unknown';
  }
  $line = implode("\t", [$ad['id'], $isuad, $ua]) . "\n";
  fputs($fp, $line, strlen($line));
  fclose($fp);

  return redirect_to_alt($ad['destination']);
});

dispatch_get('/me/report', function() {
  $advertiser_id = advertiser_id();

  if (empty($advertiser_id)) {
    halt(401);
  }

  $report = [];
  $redis = option('redis');
  $ad_keys = $redis->smembers(advertiser_key($advertiser_id));
  foreach ($ad_keys as $ad_key) {
    $ad = $redis->hgetall($ad_key);
    if (empty($ad)) {
      continue;
    }
    $imp = intval(fetch($ad, 'impressions', 0));
    $ad['impressions'] = $imp;
    $report[$ad['id']] = ['ad' => $ad, 'clicks' => 0, 'impressions' => $imp];
  }

  $logs = get_log($advertiser_id);
  foreach ($logs as $ad_id => $clicks) {
    if (!isset($report[$ad_id])) {
      $report[$ad_id] = [];
    }
    $report[$ad_id]['clicks'] = count($clicks);
  }

  return json((object)$report);
});

dispatch_get('/me/final_report', function() {
  $advertiser_id = advertiser_id();

  if (empty($advertiser_id)) {
    halt(401);
  }

  $reports = [];
  $redis = option('redis');
  $ad_keys = $redis->smembers(advertiser_key($advertiser_id));

  foreach ($ad_keys as $ad_key) {
    $ad = $redis->hgetall($ad_key);
    if (empty($ad)) {
      continue;
    }

    $imp = intval(fetch($ad, 'impressions', 0));
    $ad['impressions'] = $imp;
    $reports[$ad['id']] = ['ad' => $ad, 'clicks' => 0, 'impressions' => $imp];
  }

  $logs = get_log($advertiser_id);

  foreach ($reports as $ad_id => $report) {
    $log = fetch($logs, $ad_id, []);
    $report['clicks'] = count($log);

    $breakdown = array('gender' => (object)[], 'agents' => (object)[], 'generations' => (object)[]);
    foreach ($log as $click) {
      incr_hash($breakdown['gender'], $click['gender']);
      incr_hash($breakdown['agents'], $click['agent']);

      if (isset($click['age']) && !empty($click['age'])) {
        $generation = intval($click['age'] / 10);
      }
      else {
        $generation = 'unknown';
      }
      incr_hash($breakdown['generations'], $generation);
    }

    $report['breakdown'] = $breakdown;
    $reports[$ad_id] = $report;
  }

  return json((object)$reports);
});

dispatch_post('/initialize', function() {
  $redis = option('redis');

  $keys = $redis->keys('isu4:*');
  foreach ($keys as $key) {
    $redis->del($key);
  }

  array_map('unlink', glob(get_dir('log') . '/*'));
  content_type('text/plain');
  return 'OK';
});

run();
