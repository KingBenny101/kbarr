// Basic media info - used by search and library list endpoints
export interface Media {
    ID: number
    aid: number
    title: string
    poster_url: string
    added?: boolean
}

export interface Episode {
    ID: number
    anidb_id: string
    type: number
    ep_no: string
    title: string
    air_date: string
}

// Extended media details - used by detail endpoint
export interface MediaDetails {
    ID: number
    CreatedAt: string
    UpdatedAt: string
    title: string
    aid: number
    alternate_titles: string
    description: string
    release_date: string
    genres: string
    poster_url: string
    total_episodes: number
    total_seasons: number
    episodes: Episode[]
}