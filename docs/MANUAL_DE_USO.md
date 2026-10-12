# Manual de uso

Sistema de administración para taquería. Se usa desde **tablet** (principal), celular o
computadora, con el navegador. Cada persona entra con su propio usuario y contraseña, y
solo ve las secciones de su rol.

## 1. Entrar y moverse

1. Abre la dirección del sistema e inicia sesión.
2. En tablet/celular el menú está oculto: toca **☰** (arriba a la izquierda) para abrirlo;
   se cierra solo al elegir una sección o tocar fuera.
3. **Cerrar sesión** está al final del menú.

| Rol | Ve en el menú |
|---|---|
| admin | Todo |
| gerente | Mesas, Cocina, Cortes, Panel, Personal, Catálogo, Auditoría |
| cajero | Mesas, Pedidos / llevar, Cortes de caja |
| mesero | Mesas, Pedidos / llevar |
| cocina | Cocina (KDS) |
| contador | Cortes (consulta), Panel/KPIs, Contaduría |

## 2. Mesas y comandas (mesero, cajero, gerente, admin)

**Mapa de mesas**: las mesas se agrupan por zona (Interior, Terraza…). Colores/estados:

* **Libre** – disponible. Tócala para **abrirla**: queda a tu nombre y empieza a contar el tiempo.
* **Ocupada** – con clientes.
* **Cuenta solicitada** – los clientes ya pidieron la cuenta.

> No existe estado de "limpieza": el sistema no controla ese proceso. Cuando se cierra la
> cuenta, la mesa queda **Libre** de inmediato.

**Tomar la comanda** (dentro de la mesa):

1. Toca un producto del catálogo. Si tiene modificadores (ej. "con cebolla", "sin picante")
   elige las opciones y agrega notas.
2. Los productos van al **carrito**; revisa y toca **Enviar a cocina**. Cada envío es una *tanda*.
3. Para agregar más después, repite: se suma a la misma comanda.
4. Cancelar una partida ya enviada solo lo puede hacer **admin/gerente** y exige un **motivo**.

**Botones de la mesa**

* **Generar cuenta**: junta todo lo consumido en una cuenta (ver sección 4) y marca la mesa como
  *cuenta solicitada*. Si la cuenta ya se generó y falta cobrarla, el botón cambia a **Cobrar cuenta**
  y arriba aparece un aviso amarillo con el saldo pendiente.
* **Liberar mesa (sin consumo)**: solo aparece si la mesa no tiene comanda; sirve para una
  mesa abierta por error.

Sin internet, las comandas se guardan en el dispositivo y se envían al reconectar (aviso amarillo).

## 2.1 Pedidos para llevar y a domicilio (sin mesa)

Menú **Pedidos / llevar** (mesero, cajero, gerente, admin):

1. **+ Nuevo pedido** → elige *Para llevar* o *A domicilio*. Para domicilio, teléfono y dirección son obligatorios.
2. Elige productos por categoría y toca **Enviar a cocina** (en cocina aparece como "Para llevar"/"A domicilio").
3. El pedido queda en la lista de pendientes con el estado de cada partida. Cuando el cliente paga o recoge, toca **Cobrar**:
   se abre la cuenta y se cobra igual que en una mesa (sección 4). Al cerrar la cuenta, el pedido sale de la lista.

## 3. Cocina – KDS

Pantalla de comandas entrantes. Cada partida pasa por **enviado → preparando → listo**; se
cambia tocando el botón de la partida. Se actualiza sola cada pocos segundos.

## 4. Cobrar y cerrar la cuenta (cajero, gerente, admin)

Flujo completo, igual para cualquier rol con acceso a Mesas:

1. En la mesa toca **Generar cuenta**. Se abre la cuenta con todas las partidas.
2. *(Opcional)* **Descuento / cortesía**: solo admin/gerente, con motivo obligatorio.
3. **Cobrar**: elige método (efectivo, tarjeta débito/crédito, transferencia, vale de despensa),
   monto y propina. Puedes hacer **varios pagos** (pago mixto o cada comensal paga su parte; hay
   un ayudante para dividir en partes iguales). No se puede pagar más que el saldo.
4. Con **saldo = 0**, toca **Cerrar cuenta**.
5. **La mesa se libera sola** y puedes tocar **Volver a mesas**. Ya está lista para abrirse otra vez.

Cuentas divididas por producto: si la mesa tiene más de una cuenta, la mesa se libera cuando
**todas** estén cerradas y no queden partidas sin cobrar.

> **Importante:** generar o imprimir la cuenta **no libera** la mesa; hay que **cobrar y cerrar** la cuenta.
> Si regresas a la mesa y vuelves a tocar "Cobrar cuenta", el sistema retoma la misma cuenta (no crea otra).

Para **imprimir/reimprimir ticket** usa el botón al final de la cuenta. El ticket no tiene
validez fiscal salvo que se facture (CFDI).

## 5. Cortes de caja (cajero, gerente, admin; el contador solo consulta)

1. **Abrir turno** con el fondo inicial. Los cobros en efectivo/tarjeta se ligan a tu turno.
2. **Corte X**: parcial, para consultar sin cerrar el turno.
3. **Cerrar turno → Corte Z**: cuenta el efectivo y captúralo. El sistema calcula el
   *esperado* (fondo inicial + efectivo cobrado) y la **diferencia** (faltante/sobrante). Se puede exportar.

## 6. Panel / KPIs (admin, gerente, contador)

Indicadores del negocio por rango de fechas. Cómo se calcula cada uno: [`KPIS.md`](KPIS.md).

## 7. Personal (admin, gerente)

Tarjetas por empleado, con búsqueda y la casilla "Mostrar dados de baja".

* **Editar**: nombre, correo (su usuario), **nueva contraseña** (opcional), **rol**, puesto, **sueldo**, fecha de ingreso y si está activo.
* **Dar de baja / Reactivar**: un empleado de baja ya no puede entrar, pero su historial se conserva (lo recomendado).
* **Eliminar**: borra al empleado por completo; solo se permite si nunca ha tomado comandas, cobrado ni operado caja. Si ya tiene historial, el sistema lo impide y se debe dar de baja. No puedes darte de baja ni eliminarte a ti mismo.
* **Asistencia**: historial de entradas/salidas con duración; gerencia puede marcar entrada/salida por alguien que lo olvidó.
* **Incidencia**: registrar falta, retardo o permiso, con nota, y ver el historial.

### Checador (todos los roles)

Menú **Mi checador**: cada empleado marca su **entrada** al llegar y su **salida** al terminar; se ve cuánto lleva
trabajado y sus últimos registros. Cada quien solo puede marcar y ver lo suyo (gerencia ve a todos desde Personal).

## 8. Catálogo (admin, gerente)

Dos pestañas:

* **Productos**: categorías a la izquierda (crear, renombrar, eliminar si están vacías) y tarjetas de producto a la derecha, con búsqueda.
  Cada producto se puede **editar** (nombre, categoría, precio, qué grupos de opciones ofrece), **ocultar/mostrar** en el menú (sin perder su historial)
  y **eliminar** (solo si nunca se ha vendido; si ya se vendió, ocúltalo).
* **Modificadores**: grupos de opciones (ej. "Guisado": pastor, suadero; "Extras": queso +$10). Cada grupo puede ser obligatorio y/o de varias opciones;
  se agregan, editan y quitan opciones con su costo extra. Luego se asignan al producto al editarlo.

Los cambios de precio no alteran cuentas ya cerradas.

## 9. Zonas y mesas (admin, gerente)

En **Mesas → Administrar zonas y mesas**: crear/renombrar/eliminar zonas (solo si no tienen mesas),
y crear/editar/eliminar mesas. Eliminar una mesa la **desactiva** (no pierde historial) y se puede
**reactivar**; no se puede eliminar una mesa ocupada.

## 10. Contaduría (contador y admin)

Aquí el contador **da de alta, edita y da de baja** la información financiera:

| Pestaña | Qué se puede hacer |
|---|---|
| Reporte de ventas | Consultar por fechas, exportar CSV |
| Gastos | Registrar, editar, eliminar; exportar CSV (renta, luz, gas…) |
| Compras | Registrar, editar, eliminar compras a proveedores |
| Inventario | Crear/editar/eliminar insumos; registrar **entradas, salidas y mermas**; ver el **historial** y deshacer un movimiento (la existencia se corrige sola). En rojo = por debajo del mínimo |
| Facturación (CFDI) | Crear una factura a partir de un número de cuenta; **cancelar** (no se borra). El timbrado es **simulado** |
| Impuestos | IVA trasladado e ISR (cálculo de referencia) |
| Estado de resultados | Ingresos − gastos − compras − nómina = utilidad |

Qué NO puede hacer el contador: tomar comandas, cobrar, ni modificar catálogo, personal o mesas.
Todo alta/edición/baja queda en **Auditoría**. Si no hay fecha en un gasto/compra se usa la de hoy.

## 11. Auditoría (admin, gerente)

Bitácora de acciones sensibles: quién, qué y cuándo (descuentos, cancelaciones, altas/bajas de
contaduría, etc.).

## 12. Preguntas frecuentes

* **"Cerré la cuenta y la mesa no se liberó"**: revisa si la mesa tiene otra cuenta abierta o
  partidas nuevas sin cobrar (aparecerán al generar cuenta de nuevo).
* **"No me deja cerrar la cuenta"**: el saldo debe ser 0.
* **"Abrí una mesa por error"**: dentro de la mesa, **Liberar mesa (sin consumo)**.
* **"No veo una sección"**: tu rol no tiene acceso; pídeselo al admin.
