import redis

from flask import (
    Flask, request, redirect, url_for, jsonify,
    render_template, _app_ctx_stack, make_response
)
from werkzeug.contrib.fixers import ProxyFix

import os, hashlib, fcntl, re, mimetypes, shutil
from datetime import date

app = Flask(__name__, static_url_path='')
app.wsgi_app = ProxyFix(app.wsgi_app)

def get_dir(name):
    base_dir = '/tmp/python/'
    path = base_dir + name
    try:
        os.makedirs(path, 0755)
    except os.error:
        pass
    return path

def get_redis():
    top = _app_ctx_stack.top
    if not hasattr(top, 'redis'):
        top.redis = redis.StrictRedis(host='localhost', port=6379, db=0)
    return top.redis

def advertiser_id():
    if 'X-Advertiser-Id' in request.headers:
        return request.headers['X-Advertiser-Id']
    else:
        return None

def ad_key(slot, id):
    return 'isu4:ad:%s-%s' % (slot, id)

def asset_key(slot, id):
    return 'isu4:asset:%s-%s' % (slot, id)

def advertiser_key(id):
    return 'isu4:advertiser:%s' % (id,)

def slot_key(slot):
    return 'isu4:slot:%s' % (slot,)

def next_ad_id():
    return get_redis().incr('isu4:ad-next')

def next_ad(slot):
    key = slot_key(slot)

    id = get_redis().rpoplpush(key, key)
    if not id:
        return None

    ad = get_ad(slot, id)
    if ad:
        return ad
    else:
        get_redis().lrem(key, 0, id)
        return next_ad(slot)

def get_ad(slot, id):
    key = ad_key(slot, id)
    ad = get_redis().hgetall(key)

    if not ad:
        return None

    ad['impressions'] = int(ad['impressions'])
    ad['asset']    = url_for('route_get_ad_asset',    _external=True, slot=slot, id=id)
    ad['counter']  = url_for('route_post_ad_count',   _external=True, slot=slot, id=id)
    ad['redirect'] = url_for('route_get_ad_redirect', _external=True, slot=slot, id=id)

    return ad

def decode_user_key(id):
    if not id or id == '':
        return { 'gender': 'unknown', 'age': None }
    gender, age = [int(x) for x in id.split('/', 2)]
    if gender == 0:
        gender = 'female'
    else:
        gender = 'male'

    return { 'gender': gender, 'age': age }

def get_log_path(advr_id):
    dir = get_dir('log')
    return dir + '/' + advr_id.split('/')[-1]

def get_log(id):
    path = get_log_path(id)
    if not os.path.exists(path):
        return {}

    result = {}
    f = open(path, 'r')
    fcntl.flock(f, fcntl.LOCK_SH)
    for line in f:
        ad_id, user, agent = line.rstrip().split("\t", 3)
        if not ad_id in result:
            result[ad_id] = []

        user_attr = decode_user_key(user)
        result[ad_id].append({
            'ad_id': ad_id,
            'user': user,
            'agent': agent,
            'gender': user_attr['gender'],
            'age': user_attr['age']
            })
    f.close()

    return result

def fetch(dict, key, default_value=None):
    if key in dict:
        return dict[key]
    else:
        return default_value

def incr_dict(dict, key):
    if not key in dict:
        dict[key] = 0
    dict[key] += 1
    return dict

@app.route('/slots/<slot>/ads', methods=['POST'])
def route_post_ad(slot):
    if not advertiser_id():
        return '', 404

    asset = request.files['asset']

    id = next_ad_id()
    key = ad_key(slot, id)
    type = fetch(request.form, 'type')
    if not type:
        type = asset.mimetype
    if not type:
        type = 'video/mp4'

    redis = get_redis()
    redis.hmset(key, {
        'slot': slot,
        'id': id,
        'title': fetch(request.form, 'title'),
        'type': type,
        'advertiser': advertiser_id(),
        'destination': fetch(request.form, 'destination'),
        'impressions': 0
        })

    redis.set(asset_key(slot, id), asset.read())
    redis.rpush(slot_key(slot), id)
    redis.sadd(advertiser_key(advertiser_id()), key)

    return jsonify(get_ad(slot, id))

@app.route('/slots/<slot>/ad')
def route_get_ad(slot):
    ad = next_ad(slot)
    if ad:
        return redirect(url_for('route_get_ad_with_id', slot=slot, id=ad['id']))
    else:
        return jsonify({'error': 'not_found'}), 404

@app.route('/slots/<slot>/ads/<id>')
def route_get_ad_with_id(slot, id):
    ad = get_ad(slot, id)
    if ad:
        return jsonify(ad)
    else:
        return jsonify({'error': 'not_found'}), 404

@app.route('/slots/<slot>/ads/<id>/asset')
def route_get_ad_asset(slot, id):
    ad = get_ad(slot, id)
    if not ad:
        return jsonify({'error': 'not_found'}), 404

    redis = get_redis()
    content_type = fetch(ad, 'type', 'application/octet-stream')

    data = redis.get(asset_key(slot, id))
    if not 'Range' in request.headers:
        response = make_response(data)
        response.headers['Content-Type'] = content_type
        return response

    range = request.headers['Range']
    found = re.compile('^bytes=(\d*)-(\d*)$').findall(range)

    if not found:
        return '', 416

    head, tail = found[0]

    if head=='' and tail =='':
        return '', 416

    if head == '':
        head = 0
    else:
        head = int(head)

    if tail == '':
        tail = len(data) - 1
    else:
        tail = int(tail)

    if head < 0 or head >= len(data) or tail < 0:
        return '', 416

    range_data = data[head:(tail+1)]
    response = make_response(range_data)
    response.status_code = 206
    response.headers['Content-Type'] = content_type
    response.headers['Content-Length'] = len(range_data)
    response.headers['Content-Range'] = 'bytes %d-%d/%d' % (head, tail, len(data))

    return response

@app.route('/slots/<slot>/ads/<id>/count', methods=['POST'])
def route_post_ad_count(slot, id):
    key = ad_key(slot, id)
    redis = get_redis()

    if not redis.exists(key):
        return jsonify({'error': 'not_found'}), 404

    redis.hincrby(key, 'impressions', 1)

    return '', 204

@app.route('/slots/<slot>/ads/<id>/redirect')
def route_get_ad_redirect(slot, id):
    ad = get_ad(slot, id)

    if not ad:
        return jsonify({'error': 'not_found'}), 404

    isuad = request.cookies.get('isuad')
    if not isuad:
        isuad = ''
    ua = fetch(request.headers, 'User-Agent', 'unknown')

    path = get_log_path(ad['advertiser'])
    f = open(path, 'a')
    fcntl.flock(f, fcntl.LOCK_EX)
    f.write("\t".join([ad['id'], isuad, ua]))
    f.write("\n")

    f.close()

    return redirect(ad['destination'])

@app.route('/me/report')
def route_get_report():
    advr_id = advertiser_id()

    if not advr_id:
        return '', 401

    redis = get_redis()

    report = {}
    ad_keys = redis.smembers(advertiser_key(advr_id))
    for ad_key in ad_keys:
        ad = redis.hgetall(ad_key)
        if not ad:
            continue
        imp = int(fetch(ad, 'impressions', 0))
        ad['impressions'] = imp
        report[ad['id']] = { 'ad': ad, 'clicks': 0, 'impressions': imp }

    for ad_id, clicks in get_log(advr_id).items():
        if not ad_id in report:
            report[ad_id] = {}
        report[ad_id]['clicks'] = len(clicks)

    return jsonify(report)

@app.route('/me/final_report')
def route_get_final_report():
    advr_id = advertiser_id()

    if not advr_id:
        return '', 401

    redis = get_redis()

    reports = {}
    for ad_key in redis.smembers(advertiser_key(advr_id)):
        ad = redis.hgetall(ad_key)
        if not ad:
            continue
        imp = int(fetch(ad, 'impressions', 0))
        ad['impressions'] = imp
        reports[ad['id']] = { 'ad': ad, 'clicks': 0, 'impressions': imp }

    logs = get_log(advr_id)

    for ad_id, report in reports.items():
        log = fetch(logs, ad_id, [])
        report['clicks'] = len(log)

        breakdown = { 'gender': {}, 'agents': {}, 'generations': {} }
        for click in log:
            incr_dict(breakdown['gender'], click['gender'])
            incr_dict(breakdown['agents'], click['agent'])
            if 'age' in click and click['age'] != None:
                generation = int(click['age']) / 10
            else:
                generation = 'unknown'
            incr_dict(breakdown['generations'], generation)

        report['breakdown'] = breakdown
        reports[ad_id] = report

    return jsonify(reports)

@app.route('/initialize', methods=['POST'])
def route_post_initialize():
    redis = get_redis()

    for key in redis.keys('isu4:*'):
        redis.delete(key)

    shutil.rmtree(get_dir('log'))

    response = make_response('OK')
    response.headers['Content-Type'] = 'text/plain'
    return response

if __name__ == '__main__':
    port = int(os.environ.get('PORT', '8080'))
    app.run(debug=1, host='localhost', port=port)
