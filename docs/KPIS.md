# KPIs: cómo se clasifican y cómo se calculan

Todos los indicadores salen de **un solo endpoint** (`GET /api/v1/kpis/dashboard?from=AAAA-MM-DD&to=AAAA-MM-DD`)
y se muestran en **Panel / KPIs** (roles: admin, gerente, contador). Sin fechas, el rango es
los **últimos 7 días**. Código: `backend/internal/kpis/store.go`.

## Regla general (importante)

* Una venta **solo cuenta cuando la cuenta está cerrada**. Mesas abiertas, cuentas por
  cobrar o con saldo pendiente **no** entran.
* La fecha que se usa es la de **cierre de la cuenta** (`closed_at`), no la de la comanda.
* Las partidas **canceladas** nunca cuentan.
* Los montos de venta son **netos**: ya descontados los descuentos y cortesías.

## Clasificación

| Familia | KPI | Qué responde |
|---|---|---|
| **Ventas** | Venta total | ¿Cuánto vendí? |
| | Venta por método de pago | ¿Cómo me pagaron? |
| | Ticket promedio | ¿Cuánto deja cada mesa? |
| **Productos** | Producto más / menos vendido | ¿Qué sale y qué no? |
| **Operación** | Rotación de mesas | ¿Cuánto tiempo ocupa una mesa? |
| **Personal** | Propinas por mesero · Venta por mesero | ¿Quién vende y quién recibe propina? |
| **Caja** | Venta por turno · Faltantes/sobrantes | ¿Cuadra la caja? |

## Detalle de cada KPI

| KPI | Fórmula | Notas |
|---|---|---|
| **Venta total** | Suma de `total` de las cuentas cerradas en el rango | Neta de descuentos/cortesías. Sin propinas. |
| **Venta por método** | Suma de pagos (`efectivo`, `tarjeta_debito`, `tarjeta_credito`, `transferencia`, `vale_despensa`) de cuentas cerradas | Si hubo pago mixto, cada parte va a su método. La propina se registra aparte (`tip_amount`) y no entra aquí. |
| **Ticket promedio** | Promedio de `total` por cuenta cerrada | Una mesa con cuenta dividida en N cuentas cuenta N tickets. |
| **Más / menos vendido** | Suma de cantidades por producto (nombre guardado al momento de la venta) en cuentas cerradas; se toma el mayor / menor | El "menos vendido" solo considera productos que vendieron al menos 1 pieza (uno sin ventas no aparece). |
| **Rotación (min)** | Promedio de `cerrada_en − abierta_en` de las sesiones de mesa | La sesión se abre al **abrir la mesa** y se cierra cuando la mesa **queda libre**, lo cual ahora pasa solo al cerrar la última cuenta. Se agrupa por fecha de apertura. |
| **Propinas totales** | Suma de `tip_amount` de los pagos de cuentas cerradas | |
| **Propinas por mesero** | Propina de cada cuenta repartida **proporcionalmente** al consumo atendido por cada mesero en esa cuenta | Con un solo mesero por cuenta (lo normal) es exacto; si se traspasó la mesa se reparte por proporción. |
| **Venta por mesero** | Suma de partidas (precio × cantidad + modificadores) agrupada por mesero de la comanda | Es a **precio de lista**, antes de descuentos: sirve para comparar meseros, no para cuadrar contra Venta total. |
| **Venta por turno** | Suma de pagos ligados a cada turno de caja abierto en el rango | Solo los pagos en efectivo/tarjeta se ligan al turno del cajero que cobra. |
| **Faltantes/sobrantes** | Suma de la diferencia (`contado − esperado`) de los cortes Z del rango | Negativo = faltante (se muestra en rojo). |

## Por qué a veces "no cuadran" dos números

* **Venta total vs. Venta por mesero**: la primera es neta, la segunda a precio de lista.
* **Venta total vs. Venta por método**: deben coincidir (ambas excluyen propina); si no, hay cuentas cerradas con pagos de otra fecha.
* **Reporte de ventas (Contaduría)**: el total por día usa lo mismo que Venta total (neto); el
  desglose por producto/categoría va a precio de lista.
* **Rotación** cambió al simplificar el flujo: antes dependía de que alguien liberara la mesa
  a mano; ahora se mide hasta el cierre de la última cuenta.
