package cfgm

import (
	"maps"

	"github.com/urfave/cli/v3"
)

// options 配置加载选项。
type options struct {
	appName             string // 应用名称，用于生成默认配置路径
	cmd                 *cli.Command
	configPaths         []string
	baseDir             string // 路径基准目录，用于将相对路径转换为绝对路径
	baseDirSet          bool   // 是否显式设置了 baseDir（区分空字符串和未设置）
	envPrefix           string
	envBindings         map[string]string
	envBindKey          string
	noTemplateExpansion bool // 是否禁用配置文件模板展开（默认启用）
	callerSkip          int  // FindProjectRoot 的调用栈跳过层数（0 表示使用默认值）
}

// Option 配置加载选项函数。
type Option func(*options)

// WithCommand 设置 CLI 命令，用于从 CLI flags 加载配置。
//
// CLI flags 具有最高优先级，仅当用户明确指定时才覆盖其他配置源。
func WithCommand(cmd *cli.Command) Option {
	return func(o *options) {
		o.cmd = cmd
	}
}

// WithAppName 设置应用名称。
//
// 设置后会自动配置默认的配置文件搜索路径（如果未通过 [WithConfigPaths] 显式设置）。
// 搜索路径规则见 [DefaultPaths]。
//
// 示例：
//
//	cfgm.Load(defaultConfig,
//	    cfgm.WithAppName("myapp"),  // 自动搜索 .myapp.yaml 等
//	    cfgm.WithCommand(cmd),
//	)
func WithAppName(name string) Option {
	return func(o *options) {
		o.appName = name
	}
}

// WithConfigPaths 设置配置文件搜索路径。
//
// 按顺序搜索，找到第一个即停止。可使用 [DefaultPaths] 获取默认路径。
func WithConfigPaths(paths ...string) Option {
	return func(o *options) {
		o.configPaths = paths
	}
}

// WithBaseDir 设置相对路径的基准目录。
//
// 默认情况下，[Load] 使用项目根目录（go.mod 所在目录）作为基准。
// 使用此选项可覆盖默认行为：
//   - 传入空字符串：使用当前工作目录
//   - 传入自定义路径：使用指定目录
//
// 注意：绝对路径不受影响。
func WithBaseDir(path string) Option {
	return func(o *options) {
		o.baseDir = path
		o.baseDirSet = true
	}
}

// WithCallerSkip 设置 [FindProjectRoot] 的调用栈跳过层数。
//
// 当在封装函数中调用 [Load] 时，需要跳过更多层调用栈以正确定位项目根目录。
//
// 示例：
//
//	// 在封装函数中使用
//	func LoadMyConfig() (*Config, error) {
//	    return cfgm.Load(DefaultConfig(),
//	        cfgm.WithCallerSkip(2),  // 跳过: load → Load → LoadMyConfig
//	    )
//	}
//
// 注意：
//   - 默认值根据调用函数自动确定（Load/LoadCmd: 1, MustLoad/MustLoadCmd: 2）
//   - 每增加一层封装，skip 值需要相应增大
//   - 如果使用 [WithBaseDir] 显式设置了基准目录，此选项无效
func WithCallerSkip(skip int) Option {
	return func(o *options) {
		o.callerSkip = skip
	}
}

// WithEnvPrefix 设置环境变量前缀。
//
// 启用后，会从环境变量加载配置。优先级：配置文件 < WithEnvPrefix < WithEnvBindKey < WithEnvBindings < CLI flags。
//
// 环境变量命名规则：
//   - 前缀 + 大写的 koanf key
//   - 点号 (.) 和连字符 (-) 都转为下划线 (_)
//
// 示例 (前缀为 "MYAPP_")：
//   - MYAPP_DEBUG → debug
//   - MYAPP_SERVER_URL → server.url
//   - MYAPP_CLIENT_REV_AUTH_USER → client.rev-auth-user (支持连字符)
//
// 注意：通过反射自动生成所有 koanf key 的绑定，因此支持任意命名的 koanf key。
// 若同一配置路径被 [WithEnvBindings] 或 [WithEnvBindKey] 显式绑定，则显式绑定优先。
func WithEnvPrefix(prefix string) Option {
	return func(o *options) {
		o.envPrefix = prefix
	}
}

// WithEnvBinding 绑定单个环境变量到配置路径。
//
// 用于复用第三方工具的标准环境变量，优先级高于 WithEnvPrefix。
//
// 示例：
//
//	config.WithEnvBinding("REDIS_URL", "redis.url")
//	config.WithEnvBinding("ETCDCTL_ENDPOINTS", "etcd.endpoints")
func WithEnvBinding(envKey, configPath string) Option {
	return func(o *options) {
		if o.envBindings == nil {
			o.envBindings = make(map[string]string)
		}
		o.envBindings[envKey] = configPath
	}
}

// WithEnvBindings 批量绑定环境变量到配置路径。
//
// 用于复用第三方工具的标准环境变量，优先级高于 WithEnvPrefix。
//
// 示例：
//
//	config.WithEnvBindings(map[string]string{
//	    "REDIS_URL":         "redis.url",
//	    "ETCDCTL_ENDPOINTS": "etcd.endpoints",
//	    "MYSQL_PWD":         "database.password",
//	})
func WithEnvBindings(bindings map[string]string) Option {
	return func(o *options) {
		if o.envBindings == nil {
			o.envBindings = make(map[string]string)
		}
		maps.Copy(o.envBindings, bindings)
	}
}

// WithEnvBindKey 设置配置文件中的环境变量绑定节点名称。
//
// 启用后，会从配置文件的指定节点读取环境变量绑定关系，无需修改代码即可配置映射。
// 配置文件中的绑定优先级低于代码中的 [WithEnvBindings]（代码显式指定更优先）。
//
// 配置文件示例：
//
//	envbind:
//	  REDIS_URL: redis.url
//	  ETCDCTL_ENDPOINTS: etcd.endpoints
//
//	redis:
//	  url: "redis://localhost:6379"
func WithEnvBindKey(key string) Option {
	return func(o *options) {
		o.envBindKey = key
	}
}

// WithoutTemplateExpansion 禁用配置文件的模板展开功能。
//
// 默认情况下，配置文件会自动进行模板展开，支持以下语法：
//
//   - {{env "VAR"}} 或 {{env "VAR" "default"}} - 获取环境变量
//   - {{.VAR | default "fallback"}} - Taskfile 风格直接访问环境变量
//   - {{coalesce .VAR1 .VAR2 "default"}} - 返回第一个非空值
//
// 使用此选项可禁用模板展开，配置文件中的 {{...}} 将作为字面量保留。
func WithoutTemplateExpansion() Option {
	return func(o *options) {
		o.noTemplateExpansion = true
	}
}
