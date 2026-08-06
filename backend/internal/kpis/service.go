package kpis

import "time"

type Service struct {
	store *Store
}

func NewService(store *Store) *Service {
	return &Service{store: store}
}

func normalizeRange(from, to string) (string, string) {
	if from == "" {
		from = time.Now().AddDate(0, 0, -6).Format("2006-01-02") // últimos 7 días por defecto
	}
	if to == "" {
		to = time.Now().Format("2006-01-02")
	}
	return from, to
}

func (s *Service) Dashboard(from, to string) (Dashboard, error) {
	from, to = normalizeRange(from, to)
	d := Dashboard{From: from, To: to}

	var err error
	if d.VentaTotal, err = s.store.VentaTotal(from, to); err != nil {
		return d, err
	}
	if d.VentaPorMetodo, err = s.store.VentaPorMetodo(from, to); err != nil {
		return d, err
	}
	if d.TicketPromedio, err = s.store.TicketPromedio(from, to); err != nil {
		return d, err
	}
	if d.ProductoMasVendido, err = s.store.ProductoTop(from, to, false); err != nil {
		return d, err
	}
	if d.ProductoMenosVendido, err = s.store.ProductoTop(from, to, true); err != nil {
		return d, err
	}
	if d.RotacionMinutosProm, err = s.store.RotacionMinutosPromedio(from, to); err != nil {
		return d, err
	}
	if d.PropinasTotal, err = s.store.PropinasTotal(from, to); err != nil {
		return d, err
	}
	if d.PropinasPorMesero, err = s.store.PropinasPorMesero(from, to); err != nil {
		return d, err
	}
	if d.VentaPorMesero, err = s.store.VentaPorMesero(from, to); err != nil {
		return d, err
	}
	if d.VentaPorTurno, err = s.store.VentaPorTurno(from, to); err != nil {
		return d, err
	}
	if d.FaltantesSobrantes, err = s.store.FaltantesSobrantesAcumulado(from, to); err != nil {
		return d, err
	}
	return d, nil
}
