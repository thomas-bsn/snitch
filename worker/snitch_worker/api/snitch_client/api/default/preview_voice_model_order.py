from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.voice_model_order_preview import VoiceModelOrderPreview
from typing import cast



def _get_kwargs(
    id: str,
    *,
    model_id: str,

) -> dict[str, Any]:




    params: dict[str, Any] = {}

    params["model_id"] = model_id


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/admin/workers/{id}/voice-model-orders/preview".format(id=quote(str(id), safe=""),),
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | VoiceModelOrderPreview:
    if response.status_code == 200:
        response_200 = VoiceModelOrderPreview.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | VoiceModelOrderPreview]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    id: str,
    *,
    client: AuthenticatedClient | Client,
    model_id: str,

) -> Response[Error | VoiceModelOrderPreview]:
    """  Administrateur : ce que la confirmation d'un ordre de modèle doit montrer avant tout ordre :
    l'engine à la demande dont le modèle a besoin (famille, version, tâche), son état sur ce Worker
    (absent, en cours, prêt), le disque et la durée estimés tant qu'il reste à installer, et la cause de
    refus matériel éventuelle. Un modèle embarqué ou à adaptateur signé n'a besoin d'aucun engine à la
    demande (engine_required false). Famille inconnue de la table : 400 unsupported_engine_family, sans
    aucun ordre. Ne crée rien.

    Args:
        id (str):
        model_id (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | VoiceModelOrderPreview]
     """


    kwargs = _get_kwargs(
        id=id,
model_id=model_id,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    id: str,
    *,
    client: AuthenticatedClient | Client,
    model_id: str,

) -> Error | VoiceModelOrderPreview | None:
    """  Administrateur : ce que la confirmation d'un ordre de modèle doit montrer avant tout ordre :
    l'engine à la demande dont le modèle a besoin (famille, version, tâche), son état sur ce Worker
    (absent, en cours, prêt), le disque et la durée estimés tant qu'il reste à installer, et la cause de
    refus matériel éventuelle. Un modèle embarqué ou à adaptateur signé n'a besoin d'aucun engine à la
    demande (engine_required false). Famille inconnue de la table : 400 unsupported_engine_family, sans
    aucun ordre. Ne crée rien.

    Args:
        id (str):
        model_id (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | VoiceModelOrderPreview
     """


    return sync_detailed(
        id=id,
client=client,
model_id=model_id,

    ).parsed

async def asyncio_detailed(
    id: str,
    *,
    client: AuthenticatedClient | Client,
    model_id: str,

) -> Response[Error | VoiceModelOrderPreview]:
    """  Administrateur : ce que la confirmation d'un ordre de modèle doit montrer avant tout ordre :
    l'engine à la demande dont le modèle a besoin (famille, version, tâche), son état sur ce Worker
    (absent, en cours, prêt), le disque et la durée estimés tant qu'il reste à installer, et la cause de
    refus matériel éventuelle. Un modèle embarqué ou à adaptateur signé n'a besoin d'aucun engine à la
    demande (engine_required false). Famille inconnue de la table : 400 unsupported_engine_family, sans
    aucun ordre. Ne crée rien.

    Args:
        id (str):
        model_id (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | VoiceModelOrderPreview]
     """


    kwargs = _get_kwargs(
        id=id,
model_id=model_id,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    id: str,
    *,
    client: AuthenticatedClient | Client,
    model_id: str,

) -> Error | VoiceModelOrderPreview | None:
    """  Administrateur : ce que la confirmation d'un ordre de modèle doit montrer avant tout ordre :
    l'engine à la demande dont le modèle a besoin (famille, version, tâche), son état sur ce Worker
    (absent, en cours, prêt), le disque et la durée estimés tant qu'il reste à installer, et la cause de
    refus matériel éventuelle. Un modèle embarqué ou à adaptateur signé n'a besoin d'aucun engine à la
    demande (engine_required false). Famille inconnue de la table : 400 unsupported_engine_family, sans
    aucun ordre. Ne crée rien.

    Args:
        id (str):
        model_id (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | VoiceModelOrderPreview
     """


    return (await asyncio_detailed(
        id=id,
client=client,
model_id=model_id,

    )).parsed
