#include <linux/init.h>
#include <linux/module.h>
#include <linux/kernel.h>
#include <linux/proc_fs.h>
#include <linux/seq_file.h>
#include <linux/mm.h>
#include <linux/sysinfo.h>
#include <linux/sched/signal.h>
#include <linux/sched/cputime.h>
#include <linux/uaccess.h>
#include <linux/slab.h>
#include <linux/mm.h>

MODULE_LICENSE("GPL");
MODULE_AUTHOR("Josue David Velasquez Ixchop");
MODULE_DESCRIPTION("Sonda de Kernel - Proyecto 2 SO1");
MODULE_VERSION("1.0");

#define PROC_NAME "continfo_pr2_so1_202307705"

static struct proc_dir_entry *proc_entry;

static void obtener_cmdline(struct task_struct *task, char *buffer, size_t size)
{
    int res;
    unsigned long arg_start, arg_end, len;
    int i;

    if (!buffer || size == 0)
        return;

    buffer[0] = '\0';

    if (!task->mm) {
        snprintf(buffer, size, "%s", task->comm);
        return;
    }

    arg_start = task->mm->arg_start;
    arg_end   = task->mm->arg_end;

    if (arg_start == 0 || arg_end == 0 || arg_end <= arg_start) {
        snprintf(buffer, size, "%s", task->comm);
        return;
    }

    len = arg_end - arg_start;
    if (len >= size)
        len = size - 1;

    res = access_process_vm(task, arg_start, buffer, len, 0);
    if (res <= 0) {
        snprintf(buffer, size, "%s", task->comm);
        return;
    }

    buffer[res] = '\0';

    for (i = 0; i < res; i++) {
        if (buffer[i] == '\0')
            buffer[i] = ' ';
    }
}

static int al_leer_archivo(struct seq_file *m, void *v)
{
    struct sysinfo i;
    unsigned long total_ram_kb, free_ram_kb, used_ram_kb;
    struct task_struct *task;

    si_meminfo(&i);

    total_ram_kb = (i.totalram * i.mem_unit) / 1024;
    free_ram_kb  = (i.freeram * i.mem_unit) / 1024;
    used_ram_kb  = total_ram_kb - free_ram_kb;

    seq_printf(m, "RAM_TOTAL_KB:%lu\n", total_ram_kb);
    seq_printf(m, "RAM_FREE_KB:%lu\n", free_ram_kb);
    seq_printf(m, "RAM_USED_KB:%lu\n", used_ram_kb);

    for_each_process(task) {
        unsigned long vsz_kb = 0;
        unsigned long rss_kb = 0;
        unsigned long mem_pct = 0;
        unsigned long long cpu_time = 0;
        char cmdline[256];
        char container_hint[64];

        if (task->mm) {
            vsz_kb = (task->mm->total_vm * PAGE_SIZE) / 1024;
            rss_kb = (get_mm_rss(task->mm) * PAGE_SIZE) / 1024;

            if (total_ram_kb > 0) {
                mem_pct = (rss_kb * 100) / total_ram_kb;
            }
        }

        cpu_time = (unsigned long long)task->utime + (unsigned long long)task->stime;

        obtener_cmdline(task, cmdline, sizeof(cmdline));
        snprintf(container_hint, sizeof(container_hint), "%s", task->comm);

        seq_printf(m, "PROC:%d|%d|%s|%s|%s|%lu|%lu|%lu|%llu\n",
                task->pid,
                task->real_parent->pid,
                task->comm,
                cmdline,
                container_hint,
                vsz_kb,
                rss_kb,
                mem_pct,
                cpu_time);
    }

    return 0;
}

static int al_abrir_archivo(struct inode *inode, struct file *file)
{
    return single_open(file, al_leer_archivo, NULL);
}

static const struct proc_ops operaciones_archivo = {
    .proc_open = al_abrir_archivo,
    .proc_read = seq_read,
    .proc_lseek = seq_lseek,
    .proc_release = single_release,
};

static int __init continfo_init(void)
{
    proc_entry = proc_create(PROC_NAME, 0, NULL, &operaciones_archivo);
    if (!proc_entry) {
        printk(KERN_ERR "continfo: error al crear /proc/%s\n", PROC_NAME);
        return -ENOMEM;
    }

    printk(KERN_INFO "continfo: modulo cargado correctamente\n");
    printk(KERN_INFO "continfo: archivo /proc/%s creado\n", PROC_NAME);
    return 0;
}

static void __exit continfo_exit(void)
{
    if (proc_entry)
        proc_remove(proc_entry);

    printk(KERN_INFO "continfo: modulo removido correctamente\n");
}

module_init(continfo_init);
module_exit(continfo_exit);