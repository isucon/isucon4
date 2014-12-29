jQuery(function($) {
  var updateCurrentUser = function() {
    var isuadCookie = document.cookie.replace(/(?:(?:^|.*;\s*)isuad\s*\=\s*([^;]*).*$)|^.*$/, "$1");
    if (isuadCookie != "") {
      var values = isuadCookie.split('/', 2);

      var gender = values[0] == "0" ? "female" : "male";
      var age = values[1];

      $("#user_current p").text("gender:" + gender + " age:" + age);
    }
  };
  updateCurrentUser();

  $("#user_form_submit").click(function() {
    var str = [
      parseInt($("#user_form_gender").val(), 10).toString(),
      parseInt($("#user_form_age").val(), 10).toString()
    ].join('/');

    document.cookie = 'isuad=' + str;
    updateCurrentUser();
  });

  //////

  $("#post_testform").submit(function(e) {
    var form = $(this);
    e.preventDefault();

    var formData = new FormData(form[0]);

    form.find('input[type=submit]').prop('disabled', true);
    $.ajax({
        url: "/slots/" + $("#post_form_slot").val() + "/ads",
        type: 'POST',
        data: formData,
        async: true,
        contentType: false,
        processData: false,
        beforeSend: function (xhr) {
          xhr.setRequestHeader('X-Advertiser-Id', $("#post_form_advertiser").val());
        },
        success: function (data) {
          console.log(data);

          var href = "/slots/" + data.slot + "/ads/" + data.id;
          $("#post_result_url").html(
            $("<a>").attr('href', href).text(href)
          );

          $("#post_result_data").html($("<pre>").text(JSON.stringify(data)));
        },
        complete: function () {
          form.find('input[type=submit]').removeProp('disabled');
        }
    });
  });

  //////

  $(".report_form_show").click(function(e) {
    var path = e.target.dataset.target;
    var sectionTemplate = [
      "<h3>{{ad.title}}</h3>",
      "<p>Impressions: {{impressions}}, Clicks: {{clicks}}</p>",
      "<ul>{{#breakdowns}}",
      "<li>{{name}}<ul>{{#values}}<li>{{key}}: {{value}}</li>{{/values}}</ul></li>",
      "{{/breakdowns}}</ul>",
    ].join('');
    var breakdownHandler = function(val) {
      var ary = [];
      for(var k in val) {
        if (!val.hasOwnProperty(k)) continue;
        ary.push({"key": k, "value": val[k]});
      }
      return ary;
    };

    $("#reports").html('Loading...');
    $(".report_form_show").prop('disabled', true);
    $.ajax({
      url: path,
      async: true,
      beforeSend: function (xhr) {
        xhr.setRequestHeader('X-Advertiser-Id', $("#report_form_advertiser").val());
      },
      success: function(reports) {
        $("#reports").html("");
        for(var k in reports) {
          if(!reports.hasOwnProperty(k)) continue;

          var report = reports[k];
          var section = $("<section>").addClass('report');

          if (report.breakdown) {
            report.breakdowns = [];
            for(var k in report.breakdown) {
              if (!report.breakdown.hasOwnProperty(k)) continue;
              report.breakdowns.push({
                "name":   k,
                "values": breakdownHandler(report.breakdown[k])
              });
            }
          }
          section.html(Mustache.render(sectionTemplate, report));

          $("#reports").append(section);
        }
      },
      error: function(xhr, textStatus, error) {
        $("#reports").html($("<p>").text("Error: " + error).addClass("text-danger"));
      },
      complete: function() {
        $(".report_form_show").removeProp('disabled');
      }
    });
  })

  //////

  var showAd = function() {
    var slot = $("#display_form_slot").val();
    if (!slot || slot == "") slot = "1";

    var isuad = new IsuAd.Client(slot);

    $("#display_form_show").prop('disabled', true);
    isuad.showAd($("#ad_container"), function(ad) {
      $("#display_form_show").removeProp('disabled');
    });
  };

  showAd();
  $("#display_form_show").click(showAd);
});
