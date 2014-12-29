use FindBin;
use lib "$FindBin::Bin/extlib/lib/perl5";
use lib "$FindBin::Bin/lib";
use File::Basename;
use Plack::Builder;
use Isu4Final::Web;

my $root_dir = File::Basename::dirname(__FILE__);

my $app = Isu4Final::Web->psgi($root_dir);
builder {
    enable 'ReverseProxy';
    enable 'Static',
        path => qr!^/javascripts/!,
        root => $root_dir . '/public';
    $app;
};

