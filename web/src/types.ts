export interface Media {
    ID: number
    source: string
    source_id: string
    title: string
    poster_url: string
    added?: boolean
}

export interface Episode {
    ID: number
    source: string
    external_id: string
    type: number
    ep_no: string
    title: string
    air_date: string
}

export interface MediaDetails {
    ID: number
    CreatedAt: string
    UpdatedAt: string
    title: string
    source: string
    source_id: string
    alternate_titles: string
    description: string
    release_date: string
    genres: string
    poster_url: string
    total_episodes: number
    total_seasons: number
    episodes: Episode[]
}
