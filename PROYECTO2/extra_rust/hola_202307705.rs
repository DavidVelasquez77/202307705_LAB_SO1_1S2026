use kernel::prelude::*;

module! {
    type: HolaMundoRust,
    name: "hola_202307705",
    author: "Josue Velasquez",
    description: "Modulo extra en Rust para SO1",
    license: "GPL",
}

struct HolaMundoRust;

impl kernel::Module for HolaMundoRust {
    fn init(_module: &'static kernel::ThisModule) -> Result<Self> {
        //  Mostrar el mensaje con tu carnet
        pr_info!("Hola Mundo 202307705\n");
        Ok(HolaMundoRust)
    }
}

impl Drop for HolaMundoRust {
    fn drop(&mut self) {
        pr_info!("Modulo de Rust 202307705 descargado\n");
    }
}