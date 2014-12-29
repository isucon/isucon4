IsuAd = {};

IsuAd.Ad = (function() {
  var klass = function(data) {
    this.ad = data;
  };

  klass.prototype.generateComponent = function() {
    var video = $("<video>");

    video.attr('src', this.ad.asset);
    video.attr('type',this.ad.type);
    video.prop('controls', true);

    video.css('cursor', 'pointer');

    video.bind('canplay', this.counterFunc());
    video.bind('click', this.videoClickFunc());

    return video;
  };

  klass.prototype.videoClickFunc = function() {
    var self = this;
    return (function(e) {
      if (e.target.paused) {
        e.target.play();
      } else {
        e.target.pause();
        window.open(self.ad.redirect, '_blank');
      }
    });
  };

  klass.prototype.counterFunc = function() {
    var self = this;
    return (function(e) {
      $.post(self.ad.counter, '', function(d,s,xhr) {
        console.log("Counter posted");
      });
    });
  };

  return klass;
})();

IsuAd.Client = (function() {
  var klass = function(slot, endpoint) {
    this.endpoint = endpoint || klass.Endpoint;
    this.slot = slot;
  };

  klass.Endpoint = "";

  klass.prototype.showAd = function(elem, callback) {
    var self = this;
    var path = self.endpoint + "/slots/" + self.slot + "/ad";

    $.getJSON(path, function(adData) {
      console.log("Ad: " + adData.title);

      var ad = new IsuAd.Ad(adData);
      console.log(ad);


      $(elem).html("").append(ad.generateComponent());
      if (callback) callback(ad);
    }).fail(function() {
      if (callback) callback();
    });
  };

  return klass;
})();
