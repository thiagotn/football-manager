import type { Position } from '$lib/team-builder';

/**
 * Mapas entre o código de posição da API (gk/zag/lat/mei/ata) e o tipo
 * `Position` do team-builder, mais a chave i18n do nome da posição.
 * Compartilhado por PositionSelector, PlayerCrestCard e a página do grupo.
 */
export const API_TO_POS: Record<string, Position> = {
  gk:  'goalkeeper',
  zag: 'defender',
  lat: 'fullback',
  mei: 'midfielder',
  ata: 'forward',
};

export const POS_TO_API: Record<Position, string> = {
  goalkeeper: 'gk',
  defender:   'zag',
  fullback:   'lat',
  midfielder: 'mei',
  forward:    'ata',
};

export const POS_I18N_KEY: Record<Position, string> = {
  goalkeeper: 'position.gk',
  defender:   'position.zag',
  fullback:   'position.lat',
  midfielder: 'position.mei',
  forward:    'position.ata',
};
