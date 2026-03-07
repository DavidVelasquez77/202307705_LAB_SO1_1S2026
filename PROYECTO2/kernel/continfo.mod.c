#include <linux/module.h>
#include <linux/export-internal.h>
#include <linux/compiler.h>

MODULE_INFO(name, KBUILD_MODNAME);

__visible struct module __this_module
__section(".gnu.linkonce.this_module") = {
	.name = KBUILD_MODNAME,
	.init = init_module,
#ifdef CONFIG_MODULE_UNLOAD
	.exit = cleanup_module,
#endif
	.arch = MODULE_ARCH_INIT,
};



static const struct modversion_info ____versions[]
__used __section("__versions") = {
	{ 0x5218fe90, "single_open" },
	{ 0xc7ffe1aa, "si_meminfo" },
	{ 0x12cfb334, "seq_printf" },
	{ 0xa2152099, "init_task" },
	{ 0xd272d446, "__stack_chk_fail" },
	{ 0xb9e81daf, "proc_remove" },
	{ 0xd22cd56f, "seq_read" },
	{ 0x388dee05, "seq_lseek" },
	{ 0xae030cd0, "single_release" },
	{ 0xd272d446, "__fentry__" },
	{ 0xf8d7ac5e, "proc_create" },
	{ 0xe8213e80, "_printk" },
	{ 0xd272d446, "__x86_return_thunk" },
	{ 0x70eca2ca, "module_layout" },
};

static const u32 ____version_ext_crcs[]
__used __section("__version_ext_crcs") = {
	0x5218fe90,
	0xc7ffe1aa,
	0x12cfb334,
	0xa2152099,
	0xd272d446,
	0xb9e81daf,
	0xd22cd56f,
	0x388dee05,
	0xae030cd0,
	0xd272d446,
	0xf8d7ac5e,
	0xe8213e80,
	0xd272d446,
	0x70eca2ca,
};
static const char ____version_ext_names[]
__used __section("__version_ext_names") =
	"single_open\0"
	"si_meminfo\0"
	"seq_printf\0"
	"init_task\0"
	"__stack_chk_fail\0"
	"proc_remove\0"
	"seq_read\0"
	"seq_lseek\0"
	"single_release\0"
	"__fentry__\0"
	"proc_create\0"
	"_printk\0"
	"__x86_return_thunk\0"
	"module_layout\0"
;

MODULE_INFO(depends, "");


MODULE_INFO(srcversion, "2F83557EEF58E55090CAA5E");
