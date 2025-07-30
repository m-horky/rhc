project('rhc', version : '0.3.2', meson_version : '>= 0.56.0')

go = find_program('go')
systemd = dependency('systemd', version: '>=239')
bash_completion = dependency('bash-completion')

if get_option('vendor')
  meson.add_dist_script(join_paths('scripts', 'vendor.sh'))
endif

goldflags = get_option('goldflags')
goldflags += ' -X "github.com/redhatinsights/rhc/pkg/config.Version=' + meson.project_version() + '"'
goldflags += ' -X "github.com/redhatinsights/rhc/pkg/config.ShortName=rhc"'
goldflags += ' -X "github.com/redhatinsights/rhc/pkg/config.LongName=rhc"'
goldflags += ' -X "github.com/redhatinsights/rhc/pkg/config.PrefixDir=' + get_option('prefix') + '"'
goldflags += ' -X "github.com/redhatinsights/rhc/pkg/config.BinDir=' + get_option('bindir') + '"'
goldflags += ' -X "github.com/redhatinsights/rhc/pkg/config.SbinDir=' + get_option('sbindir') + '"'
goldflags += ' -X "github.com/redhatinsights/rhc/pkg/config.LibexecDir=' + get_option('libexecdir') + '"'
goldflags += ' -X "github.com/redhatinsights/rhc/pkg/config.SysconfDir=' + get_option('sysconfdir') + '"'
goldflags += ' -X "github.com/redhatinsights/rhc/pkg/config.DataDir=' + get_option('datadir') + '"'
goldflags += ' -X "github.com/redhatinsights/rhc/pkg/config.DatarootDir=' + get_option('datadir') + '"'
goldflags += ' -X "github.com/redhatinsights/rhc/pkg/config.ManDir=' + get_option('mandir') + '"'
goldflags += ' -X "github.com/redhatinsights/rhc/pkg/config.DocDir=' + get_option('prefix') / 'doc' + '"'
goldflags += ' -X "github.com/redhatinsights/rhc/pkg/config.LocalstateDir=' + get_option('localstatedir') + '"'
goldflags += ' -X "github.com/redhatinsights/rhc/pkg/config.ServiceName=yggdrasil"'

gobuildflags = get_option('gobuildflags')

rhc = custom_target(
  'rhc',
  build_always_stale: true,
  output: 'rhc',
  command: [
    go,
    'build',
    gobuildflags,
    '-o', '@OUTPUT@',
    '-ldflags', goldflags,
    'github.com/redhatinsights/rhc/cmd/rhc',
  ],
  install: true,
  install_dir: get_option('bindir'),
)

subdir('data')
subdir('dist')
subdir('doc')

summary(
  {
    'vendor': get_option('vendor'),
  },
  section: 'Options',
)
