#include <linux/init.h>
#include <linux/module.h>
#include <linux/kernel.h>
#include <linux/proc_fs.h>
#include <linux/seq_file.h>
#include <linux/mm.h>
#include <linux/sysinfo.h>
#include <linux/sched/signal.h>

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

    seq_printf(m, "=== Memoria del Sistema ===\n");
    seq_printf(m, "Total RAM: %lu KB\n", total_ram_kb);
    seq_printf(m, "Free RAM: %lu KB\n", free_ram_kb);
    seq_printf(m, "Used RAM: %lu KB\n\n", used_ram_kb);

    seq_printf(m, "=== Procesos ===\n");
    seq_printf(m, "PID\tPPID\tNOMBRE\t\tVSZ(KB)\tRSS(KB)\n");

    for_each_process(task) {
        unsigned long vsz_kb = 0;
        unsigned long rss_kb = 0;

        if (task->mm) {
            vsz_kb = (task->mm->total_vm * PAGE_SIZE) / 1024;
            rss_kb = (get_mm_rss(task->mm) * PAGE_SIZE) / 1024;
        }

        seq_printf(m, "%d\t%d\t%s\t\t%lu\t%lu\n",
                   task->pid,
                   task->real_parent->pid,
                   task->comm,
                   vsz_kb,
                   rss_kb);
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