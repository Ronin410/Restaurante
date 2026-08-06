import { cloneElement, useId, type CSSProperties, type ReactElement } from 'react'

/**
 * Envuelve un control de formulario asociando su <label> vía htmlFor/id
 * generado con useId(), para que sea accesible (lectores de pantalla,
 * click en la etiqueta) y para que las pruebas puedan ubicarlo con
 * getByLabelText. Antes cada página escribía <label> e <input> como
 * hermanos sin relación, lo que rompía ambas cosas.
 */
export function Field({
  label,
  children,
  style,
}: {
  label: string
  children: ReactElement<{ id?: string }>
  style?: CSSProperties
}) {
  const id = useId()
  return (
    <div className="field" style={style}>
      <label htmlFor={id}>{label}</label>
      {cloneElement(children, { id })}
    </div>
  )
}
