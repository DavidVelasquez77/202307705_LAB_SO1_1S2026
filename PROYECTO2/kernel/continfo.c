#include <linux/init.h>
#include <linux/module.h>
#include <linux/kernel.h>
#include <linux/proc_fs.h>
#include <linux/seq_file.h>
#include <linux/mm.h>
#include <linux/sysinfo.h>
#include <linux/sched/signal.h>
#include <linux/sched/cputime.h>

MODULE_LICENSE("GPL");
MODULE_AUTHOR("Josue David Velasquez Ixchop");
MODULE_DESCRIPTION("Sonda de Kernel - Proyecto 2 SO1");
MODULE_VERSION("1.0");

#define PROC_NAME "continfo_pr2_so1_202307705"

static struct proc_dir_entry *proc_entry;

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

        if (task->mm) {
            vsz_kb = (task->mm->total_vm * PAGE_SIZE) / 1024;
            rss_kb = (get_mm_rss(task->mm) * PAGE_SIZE) / 1024;

            if (total_ram_kb > 0) {
                mem_pct = (rss_kb * 100) / total_ram_kb;
            }
        }

        cpu_time = (unsigned long long)task->utime + (unsigned long long)task->stime;

        seq_printf(m, "PROC:%d|%d|%s|%lu|%lu|%lu|%llu\n",
                   task->pid,
                   task->real_parent->pid,
                   task->comm,
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