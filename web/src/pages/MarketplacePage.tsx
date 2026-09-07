/*
 * Copyright 2025 coze-dev Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { useTranslation } from 'react-i18next'
import { Search, Store, Copy, X, Users } from 'lucide-react'
import { marketplaceApi, type CozeProduct } from '@/lib/coze-api'
import { cn } from '@/lib/cn'

export default function MarketplacePage() {
  const { t } = useTranslation()
  const qc = useQueryClient()

  const [search, setSearch] = useState('')
  const [category, setCategory] = useState('')
  const [selectedProduct, setSelectedProduct] = useState<CozeProduct | null>(null)

  // Categories
  const { data: catData } = useQuery({
    queryKey: ['marketplace-categories'],
    queryFn: () => marketplaceApi.categories(),
  })
  const categories = catData?.categories || []

  // Product list (by category)
  const { data: productData, isLoading } = useQuery({
    queryKey: ['marketplace-products', category],
    queryFn: () => marketplaceApi.list({ category: category || undefined }),
  })
  const products = productData?.products || []

  // Search results
  const { data: searchData, isLoading: isSearching } = useQuery({
    queryKey: ['marketplace-search', search],
    queryFn: () => marketplaceApi.search(search),
    enabled: search.trim().length > 0,
  })
  const searchResults = searchData || []

  const displayProducts = search.trim() ? searchResults : products
  const displayLoading = search.trim() ? isSearching : isLoading

  // Duplicate / Import
  const importMut = useMutation({
    mutationFn: (id: string) => marketplaceApi.duplicate(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['marketplace-products'] })
      toast.success(t('marketplace.imported', 'Product imported'))
      setSelectedProduct(null)
    },
    onError: (e: Error) => toast.error(e.message),
  })

  return (
    <div className="h-full flex flex-col">
      {/* Header */}
      <div className="p-6 border-b border-gray-200 bg-white">
        <div className="flex items-center justify-between mb-4">
          <h1 className="text-2xl font-bold text-gray-900 flex items-center gap-2">
            <Store className="w-6 h-6 text-indigo-500" />
            {t('marketplace.title', 'Marketplace')}
          </h1>
        </div>

        {/* Search bar */}
        <div className="relative max-w-md mb-4">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
          <input value={search} onChange={(e) => setSearch(e.target.value)}
            placeholder={t('marketplace.searchPlaceholder', 'Search products...')}
            className="w-full pl-10 pr-10 py-2.5 border border-gray-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent" />
          {search && (
            <button onClick={() => setSearch('')}
              className="absolute right-3 top-1/2 -translate-y-1/2 p-0.5 text-gray-400 hover:text-gray-600 rounded">
              <X className="w-4 h-4" />
            </button>
          )}
        </div>

        {/* Category filter */}
        {categories.length > 0 && (
          <div className="flex gap-2 flex-wrap">
            <button onClick={() => setCategory('')}
              className={cn('px-3 py-1.5 text-xs font-medium rounded-full transition-colors',
                !category ? 'bg-blue-600 text-white' : 'bg-gray-100 text-gray-600 hover:bg-gray-200')}>
              {t('marketplace.all', 'All')}
            </button>
            {categories.map((c) => (
              <button key={c.id} onClick={() => setCategory(c.id)}
                className={cn('px-3 py-1.5 text-xs font-medium rounded-full transition-colors',
                  category === c.id ? 'bg-blue-600 text-white' : 'bg-gray-100 text-gray-600 hover:bg-gray-200')}>
                {c.name}
              </button>
            ))}
          </div>
        )}
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto p-6">
        {/* Loading */}
        {displayLoading && (
          <div className="flex items-center gap-2 text-sm text-gray-500">
            <svg className="h-4 w-4 animate-spin" viewBox="0 0 24 24" fill="none">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z" />
            </svg>
            {t('common.loading', 'Loading...')}
          </div>
        )}

        {/* Empty */}
        {!displayLoading && displayProducts.length === 0 && (
          <div className="flex flex-col items-center justify-center py-20 text-gray-400">
            <Store className="h-12 w-12 mb-3 opacity-40" />
            <p className="text-sm">
              {search.trim()
                ? t('marketplace.noResults', 'No products found for "{{q}}"', { q: search })
                : t('marketplace.empty', 'Marketplace is empty')}
            </p>
          </div>
        )}

        {/* Product grid */}
        {!displayLoading && displayProducts.length > 0 && (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
            {displayProducts.map((product) => (
              <div key={product.product_id}
                onClick={() => setSelectedProduct(product)}
                className="group bg-white rounded-lg border border-gray-200 p-4 cursor-pointer transition-all hover:shadow-md hover:border-gray-300">
                <div className="flex items-center gap-3 mb-3">
                  {product.icon_url ? (
                    <img src={product.icon_url} alt={product.name} className="h-10 w-10 rounded-lg object-cover" />
                  ) : (
                    <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-gradient-to-br from-blue-500 to-violet-500 text-white shrink-0">
                      <Store className="h-5 w-5" />
                    </div>
                  )}
                  <div className="min-w-0 flex-1">
                    <h3 className="text-sm font-medium text-gray-900 truncate group-hover:text-blue-600 transition-colors">
                      {product.name}
                    </h3>
                    {product.category && (
                      <span className="inline-block mt-0.5 rounded-full bg-gray-100 px-2 py-0.5 text-[10px] font-medium text-gray-500">
                        {product.category}
                      </span>
                    )}
                  </div>
                </div>
                {product.description && (
                  <p className="text-xs text-gray-500 line-clamp-2 mb-3">{product.description}</p>
                )}
                <div className="flex items-center gap-1 text-xs text-gray-400">
                  <Users className="h-3 w-3" />
                  {product.use_count ?? 0} {t('marketplace.uses', 'uses')}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Product Detail Modal */}
      {selectedProduct && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
          <div className="bg-white rounded-xl shadow-xl p-6 w-[480px] max-h-[80vh] overflow-y-auto">
            <div className="flex items-start justify-between mb-4">
              <div className="flex items-center gap-4">
                {selectedProduct.icon_url ? (
                  <img src={selectedProduct.icon_url} alt={selectedProduct.name} className="h-14 w-14 rounded-xl object-cover" />
                ) : (
                  <div className="flex h-14 w-14 items-center justify-center rounded-xl bg-gradient-to-br from-blue-500 to-violet-500 text-white">
                    <Store className="h-7 w-7" />
                  </div>
                )}
                <div>
                  <h2 className="text-lg font-semibold text-gray-900">{selectedProduct.name}</h2>
                  {selectedProduct.category && (
                    <span className="inline-block mt-1 rounded-full bg-gray-100 px-2.5 py-0.5 text-xs font-medium text-gray-600">
                      {selectedProduct.category}
                    </span>
                  )}
                </div>
              </div>
              <button onClick={() => setSelectedProduct(null)}
                className="p-1 text-gray-400 hover:text-gray-600 rounded">
                <X className="w-5 h-5" />
              </button>
            </div>

            {selectedProduct.description && (
              <div className="mb-4">
                <label className="block text-xs font-medium text-gray-500 mb-1">
                  {t('common.description', 'Description')}
                </label>
                <p className="text-sm text-gray-700">{selectedProduct.description}</p>
              </div>
            )}

            <div className="flex items-center gap-4 text-sm text-gray-500 mb-4">
              <span className="flex items-center gap-1">
                <Users className="h-4 w-4" />
                {selectedProduct.use_count ?? 0} {t('marketplace.uses', 'uses')}
              </span>
            </div>

            <div className="flex gap-3 pt-2 border-t border-gray-100">
              <button
                onClick={() => importMut.mutate(selectedProduct.product_id)}
                disabled={importMut.isPending}
                className="flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 disabled:opacity-50">
                <Copy className="w-4 h-4" />
                {importMut.isPending ? '...' : t('marketplace.import', 'Import')}
              </button>
              <button onClick={() => setSelectedProduct(null)}
                className="px-4 py-2 text-sm text-gray-700 bg-gray-100 rounded-lg hover:bg-gray-200">
                {t('common.close', 'Close')}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
