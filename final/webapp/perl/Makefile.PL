use ExtUtils::MakeMaker;

WriteMakefile(
    NAME          => 'Isu4Final',
    VERSION_FROM  => 'lib/Isu4Final.pm',
    PREREQ_PM     => {
        'Kossy' =>  '0.38',
    },
    MIN_PERL_VERSION => '5.008001'
);

