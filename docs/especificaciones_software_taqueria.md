# Especificaciones Funcionales — Software de Administración para Taquería

## 1. Objetivo del sistema
Desarrollar un sistema (POS + back office) para gestionar la operación diaria de una taquería: toma de comandas, control de mesas, cobro, cortes de caja, personal y contabilidad, con acceso por roles y reportes en tiempo real.

---

## 2. Módulo de Comandas

**Funciones:**
- Captura de pedido por mesa, para llevar o a domicilio.
- Catálogo de productos por categoría (tacos, órdenes, bebidas, extras, salsas).
- Modificadores: guisado, cantidad, "sin cebolla", "con todo", término de cocción, etc.
- Envío automático a cocina (impresora térmica o pantalla KDS - Kitchen Display System).
- Estatus de comanda: enviada / en preparación / lista / entregada.
- Posibilidad de agregar productos a una comanda ya abierta sin duplicar ticket.
- Cancelación o modificación de partidas con motivo y autorización de supervisor (control de mermas/fraude).
- Comandas separadas por "tanda" cuando se agregan productos después de la primera entrega.

**Datos a capturar:** mesero, mesa/pedido, hora de apertura, productos, cantidad, precio, notas, hora de envío a cocina, hora de entrega.

---

## 3. Módulo de Mesas

**Funciones:**
- Mapa visual del salón (editable: número de mesas, capacidad, zonas — interior/terraza).
- Estados de mesa: libre / ocupada / cuenta solicitada / en limpieza.
- Asignación de mesero por mesa.
- Unión de mesas (grupos grandes) y división de mesas (una mesa, varias cuentas).
- Traspaso de mesa entre meseros (con registro de quién autorizó).
- Tiempo de ocupación visible (para rotación de mesas).

---

## 4. Módulo de Cuentas y Cobro

**Funciones:**
- Generación de cuenta por mesa, agrupando todas las comandas.
- División de cuenta: por persona, por producto, o monto libre.
- Métodos de pago: efectivo, tarjeta (débito/crédito), transferencia, vales de despensa; y pagos mixtos.
- Registro de propina (efectivo y tarjeta) separada del total de venta.
- Aplicación de descuentos y cortesías (con autorización y motivo obligatorio).
- Cierre de cuenta y generación de ticket/factura.
- Reimpresión de ticket.

---

## 5. Módulo de Cortes de Caja

**Funciones:**
- Corte por turno y por cajero (apertura y cierre de caja con fondo inicial).
- Corte general del día (Z) y cortes parciales (X).
- Desglose por: forma de pago, mesero, categoría de producto, propinas, descuentos/cortesías aplicadas.
- Comparativo de venta esperada (sistema) vs. efectivo contado (control de faltantes/sobrantes).
- Historial de cortes consultable por fecha/rango de fechas.
- Exportación de cortes (PDF/Excel) para contabilidad.

---

## 6. Módulo de Administración de Personal

**Funciones:**
- Alta de empleados: datos personales, puesto, sueldo/comisión, fecha de ingreso.
- Roles y permisos: administrador, gerente, cajero, mesero, cocina.
- Control de asistencia (entrada/salida, con reloj checador digital u opción biométrica).
- Turnos y horarios asignados.
- Registro de propinas por mesero (si se reparten individualmente) o fondo común.
- Historial de ventas por mesero (para evaluar desempeño/comisiones).
- Incidencias: faltas, retardos, permisos.

---

## 7. Módulo de Contaduría

**Funciones:**
- Reportes de ventas: diario, semanal, mensual, por producto, por categoría.
- Registro de gastos e insumos (compras a proveedores, gastos operativos).
- Control básico de inventario (opcional pero recomendado): mermas, existencias de insumos clave.
- Facturación electrónica CFDI 4.0 (integración con PAC autorizado por el SAT) para clientes que requieran factura.
- Reporte de impuestos (IVA trasladado, ISR retenido si aplica).
- Estado de resultados simplificado (ingresos - gastos - nómina).
- Exportación de información contable en formatos compatibles (XML, Excel, CSV) para el contador externo.
- Acceso de solo lectura para el contador (usuario con permisos limitados).

---

## 8. Roles y Permisos de Usuario

| Rol | Accesos |
|---|---|
| Administrador/Dueño | Acceso total a todos los módulos y reportes |
| Gerente | Cortes, personal, mesas, autorización de descuentos/cancelaciones |
| Cajero | Cobro, cortes de su turno, cuentas |
| Mesero | Captura de comandas, estado de mesas |
| Cocina | Vista de comandas entrantes (KDS), marcar como listo |
| Contador | Solo lectura: reportes financieros, facturación, gastos |

---

## 9. Requerimientos Técnicos

- **Arquitectura:** aplicación web o híbrida (funcionar en tablet/POS y celular), con base de datos en la nube para consultar desde cualquier lugar.
- **Modo offline:** operación básica (comandas y cobro) aun sin internet, sincronizando al recuperar conexión.
- **Hardware sugerido:** terminal/tablet en caja, impresora térmica para tickets, impresora o pantalla en cocina, lector opcional de tarjetas.
- **Seguridad:** login individual por empleado, bitácora de acciones (quién canceló, descontó, etc.), respaldo automático de información.
- **Escalabilidad:** posibilidad de agregar sucursales a futuro bajo un mismo panel de administración.

---

## 10. Reportes y KPIs sugeridos
- Venta total y por método de pago (diario/semanal/mensual).
- Ticket promedio por mesa.
- Producto más y menos vendido.
- Rotación de mesas (tiempo promedio de ocupación).
- Propinas totales y por mesero.
- Comparativo de ventas entre turnos o entre meseros.
- Faltantes/sobrantes de caja acumulados.

---

## Notas para el desarrollo
Este documento es la base funcional. Antes de programar se recomienda definir:
1. Número de sucursales (¿una o varias?).
2. Si se requiere control de inventario detallado (costeo de recetas) o solo básico.
3. Si se necesita facturación electrónica desde el día uno o se puede agregar después.
4. Presupuesto y si se prefiere desarrollo a medida vs. adaptar un POS existente (ej. Poster, Loyverse, Square) con estas especificaciones como checklist de evaluación.
