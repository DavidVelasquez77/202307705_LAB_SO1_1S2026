obj-m := hola_202307705.o

KDIR := /lib/modules/$(shell uname -r)/build
PWD := $(shell pwd)

# Forzar la ruta de rustc si no lo encuentra el shell de root
RUSTC := $(shell which rustc)

all:
	$(MAKE) -C $(KDIR) M=$(PWD) RUSTC=$(RUSTC) HOSTRUSTC=$(RUSTC) modules

clean:
	$(MAKE) -C $(KDIR) M=$(PWD) clean